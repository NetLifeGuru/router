package router

import (
	"sync"
)

type Par struct {
	Key   string
	Value string
}

type Seg struct {
	Value string
}

type Context struct {
	Params   []Par
	Segments []Seg
	Data     map[string]interface{}
}

func (c *Context) Set(key string, value interface{}) {
	if c.Data == nil {
		c.Data = make(map[string]interface{}, 4)
	}
	c.Data[key] = value
}

func (c *Context) Get(key string) interface{} {
	if c.Data == nil {
		return nil
	}
	return c.Data[key]
}

func (c *Context) Param(key string) string {
	for _, p := range c.Params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

func (c *Context) reset() {
	c.Params = c.Params[:0]
	c.Segments = c.Segments[:0]

	if c.Data != nil {
		c.Data = nil
	}
}

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			Params:   make([]Par, 0, 8),
			Segments: make([]Seg, 0, 8),
		}
	},
}

func GetContext() *Context {
	ctx := contextPool.Get().(*Context)
	ctx.reset()
	return ctx
}

func PutContext(ctx *Context) {
	contextPool.Put(ctx)
}
