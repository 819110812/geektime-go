package web

import (
	"testing"
)

// 这里放着端到端测试的代码

func TestServer(t *testing.T) {
	s := NewHTTPServer()
	s.Get("/", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, world"))
	})
	s.Get("/user", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, user"))
	})
	s.Get("/user/home", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, user/home"))
	})
	s.Get("/user/:id", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello, user/:id"))
	})

	s.Start(":8081")
}
