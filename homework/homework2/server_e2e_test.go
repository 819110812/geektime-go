package web

import (
	"fmt"
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
	}).Use(func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)
			ctx.Resp.Write([]byte("this is first middleware"))
		}
	}).Use(func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)
			ctx.Resp.Write([]byte("this second middleware"))
		}
	})

	s.Post("/form", func(ctx *Context) {
		err := ctx.Req.ParseForm()
		if err != nil {
			fmt.Println(err)
		}
	})

	s.Start(":8081")
}
