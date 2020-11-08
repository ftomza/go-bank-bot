package bot

import (
	"context"
	"errors"
)

type StepFn func(ctx context.Context, sess *Session) error

type Step interface {
	Do(ctx context.Context, sess *Session) (Step, error)
}

type BaseStep struct {
	Fn StepFn
}

func (s BaseStep) Do(ctx context.Context, sess *Session) (Step, error) {
	return nil, s.Fn(ctx, sess)
}

func NewStep(fn StepFn) *BaseStep {
	return &BaseStep{Fn: fn}
}

type NextStep struct {
	BaseStep
	Next Step
}

func NewNextStep(fn StepFn, next Step) *NextStep {
	return &NextStep{BaseStep: *NewStep(fn), Next: next}
}

func (s NextStep) Do(ctx context.Context, sess *Session) (Step, error) {
	err := s.Fn(ctx, sess)
	if err != nil {
		return nil, err
	}
	return s.Next, nil
}

type ConditionFn func(ctx context.Context, sess *Session) (bool, error)

type ConditionalStep struct {
	ConditionFn ConditionFn
	TrueStep    Step
	FalseStep   Step
}

func NewConditionalStep(conditionFn ConditionFn, trueStep Step, falseStep Step) *ConditionalStep {
	return &ConditionalStep{ConditionFn: conditionFn, TrueStep: trueStep, FalseStep: falseStep}
}

func (s ConditionalStep) Do(ctx context.Context, sess *Session) (Step, error) {
	if ok, err := s.ConditionFn(ctx, sess); err != nil {
		return nil, err
	} else if ok {
		return s.TrueStep, nil
	}
	return s.FalseStep, nil
}

type Session struct {
	ctx  context.Context
	step Step
}

func (s *Session) AddValue(key, val interface{}) {
	s.ctx = context.WithValue(s.ctx, key, val)
}

func (s *Session) Value(key interface{}) interface{} {
	return s.ctx.Value(key)
}

func (s *Session) Run(ctx context.Context) error {
	if s.step == nil {
		return errors.New("session: nothing to do")
	}

	type resp struct {
		step Step
		err  error
	}

	ch := make(chan resp)

	go func() {
		step, err := s.step.Do(ctx, s)
		if err != nil {
			ch <- resp{
				err: err,
			}
			return
		}
		ch <- resp{
			step: step,
		}
	}()

	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-ch:
		s.step = resp.step
		return resp.err
	}

}

func NewSession(ctx context.Context, step Step) Session {
	return Session{
		ctx:  ctx,
		step: step,
	}
}
