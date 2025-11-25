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
	Data     map[string]any
	Entries  []RouteEntry

	paramMap map[string]string
	aborted  bool
}

func (c *Context) Abort() {
	c.aborted = true
}

func (c *Context) Aborted() bool {
	return c.aborted
}

func (c *Context) Set(key string, value any) {
	if c.Data == nil {
		c.Data = make(map[string]any, 4)
	}
	c.Data[key] = value
}

func (c *Context) Get(key string) any {
	if c.Data == nil {
		return nil
	}
	return c.Data[key]
}

func (c *Context) SetParams() {
	if len(c.Params) > 0 {
		return
	}

	if len(c.Entries) == 0 {
		return
	}

	entry := c.Entries[0]

	if cap(c.Params) < len(entry.Patterns) {
		c.Params = make([]Par, 0, len(entry.Patterns))
	} else {
		c.Params = c.Params[:0]
	}

	for depth, p := range entry.Patterns {
		if p.Type == _STRING {
			continue
		}
		if depth >= len(c.Segments) {
			break
		}

		segment := c.Segments[depth].Value
		c.Params = append(c.Params, Par{
			Key:   p.Slug,
			Value: segment,
		})
	}
}

func (c *Context) Param(key string) (string, bool) {
	m := c.ParamMap()
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func (c *Context) ParamMap() map[string]string {
	if c.paramMap != nil {
		return c.paramMap
	}

	c.SetParams()

	if len(c.Params) == 0 {
		c.paramMap = map[string]string{}
		return c.paramMap
	}

	m := make(map[string]string, len(c.Params))
	for _, p := range c.Params {
		m[p.Key] = p.Value
	}
	c.paramMap = m

	return c.paramMap
}

func (c *Context) reset() {
	c.aborted = false
	c.paramMap = nil

	if cap(c.Params) > 1024 {
		c.Params = make([]Par, 0, 8)
	} else {
		c.Params = c.Params[:0]
	}

	if cap(c.Segments) > 1024 {
		c.Segments = make([]Seg, 0, 8)
	} else {
		c.Segments = c.Segments[:0]
	}

	if cap(c.Entries) > 1024 {
		c.Entries = make([]RouteEntry, 0, 8)
	} else {
		c.Entries = c.Entries[:0]
	}

	c.Data = nil
}

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			Params:   make([]Par, 0, 8),
			Segments: make([]Seg, 0, 8),
			Data:     make(map[string]any, 4),
			Entries:  make([]RouteEntry, 0, 8),
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
