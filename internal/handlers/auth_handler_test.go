package handlers_test

import (
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type fakeAuthSvc struct {
	registerFn func(login, password string) (uuid.UUID, error)
	loginFn    func(login, password string) (string, models.User, error)
}

func (f *fakeAuthSvc) Register(ctx any, login, password string) (uuid.UUID, error) { // not used
	return uuid.Nil, nil
}

type authAdapter struct{ *fakeAuthSvc }

func (a authAdapter) Register(ctx interface{}, login, password string) (uuid.UUID, error) {
	return a.fakeAuthSvc.registerFn(login, password)
}
func (a authAdapter) Login(ctx interface{}, login, password string) (string, models.User, error) {
	return a.fakeAuthSvc.loginFn(login, password)
}
