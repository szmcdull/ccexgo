package rpc

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type (
	testStream struct {
		result []int
		lastID ID
		closed bool
		cond   *sync.Cond
		wait   bool
	}
)

var (
	testErr = errors.New("test error")
)

func (ts *testStream) Read() (Response, error) {
	//wait Write
	ts.cond.L.Lock()
	defer ts.cond.L.Unlock()
	defer func() { ts.wait = true }()
	for ts.wait {
		ts.cond.Wait()
	}

	if len(ts.result) == 0 {
		return nil, testErr
	}

	val := ts.result[0]
	ts.result = ts.result[1:]
	return &Result{ID: ts.lastID, Params: val}, nil
}

func (ts *testStream) Write(req Request) error {
	if ts.closed {
		return NewStreamError(errors.New("closed"))
	}
	call := req.(*Call)
	ts.lastID = call.id
	ts.wait = false
	ts.cond.Broadcast()
	return nil
}
func (ts *testStream) Close() error {
	ts.closed = true
	return nil
}

func TestCall(t *testing.T) {
	stream := &testStream{
		result: []int{1, 2, 3, 4},
		closed: false,
		cond:   sync.NewCond(&sync.Mutex{}),
		wait:   true,
	}

	conn := NewConn(stream)
	result := &Result{}
	ctx := context.Background()
	conn.Run(ctx, nil)
	for i := 1; i < 5; i++ {
		conn.Call(ctx, "", nil, result)
		if result.Result.(int) != i {
			t.Errorf("bad value %v", result.Result)
		}
	}
	//read error ctx will timeout
	ctx, _ = context.WithTimeout(ctx, time.Millisecond*100)
	conn.Call(ctx, "", nil, result)
	conn.Close()
	if err := conn.Call(ctx, "", nil, result); !errors.Is(err, &StreamError{}) {
		t.Errorf("bad expect error %v", err)
	}
	c := conn.(*connection)
	if len(c.pending) != 0 {
		t.Errorf("bad state for connection %v", c.pending)
	}
}

type (
	testStreamH struct {
		testStream
	}
)

func (tsh *testStreamH) Read() (Response, error) {
	return nil, NewStreamError(errors.New("test stream handler quit"))
}

func TestHadleMessagQuit(t *testing.T) {
	stream := &testStreamH{
		testStream: testStream{
			result: []int{},
			closed: false,
			cond:   sync.NewCond(&sync.Mutex{}),
			wait:   true,
		},
	}

	conn := NewConn(stream)
	result := &Result{}
	ctx := context.Background()
	conn.Run(ctx, nil)

	if err := conn.Call(ctx, "", nil, result); err != ErrClear {
		t.Errorf("bad error %v", err)
	}
}
