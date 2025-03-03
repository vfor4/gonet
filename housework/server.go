package housework

import (
	"context"
	"fmt"
	"sync"
)

type Rosie struct {
	mu     sync.Mutex
	chores []*Chore
}

func (r *Rosie) Add(_ context.Context, chores *Chores) (*Response, error) {

	r.mu.Lock()
	r.chores = append(r.chores, chores.Chores...)
	r.mu.Unlock()
	return &Response{Message: "ok"}, nil
}

func (r *Rosie) Complete(_ context.Context, req *CompleteRequest) (*Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.chores == nil && req.ChoreNumber < 1 || int(req.ChoreNumber) > len(r.chores) {
		return nil, fmt.Errorf("chore %d not found", req.ChoreNumber)
	}
	r.chores[req.ChoreNumber-1].Complete = true
	return &Response{Message: "ok"}, nil
}

func (r *Rosie) List(_ context.Context, _ *Empty) (*Chores, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.chores == nil {
		r.chores = make([]*Chore, 0)
	}
	return &Chores{Chores: r.chores}, nil
}
