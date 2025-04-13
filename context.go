package router

import (
	"fmt"
	"sync"
)

type Context struct {
	Params map[string]interface{}
	data   map[string]interface{}
}

func (c *Context) Set(key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}

func (c *Context) Get(key string) interface{} {
	if c.data == nil {
		return nil
	}
	return c.data[key]
}

func (c *Context) Param(key string) string {
	if val, ok := c.Params[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			Params: make(map[string]interface{}),
			data:   make(map[string]interface{}),
		}
	},
}

func (c *Context) reset() {
	for k := range c.Params {
		delete(c.Params, k)
	}
	if c.data == nil {
		c.data = make(map[string]interface{})
	} else {
		for k := range c.data {
			delete(c.data, k)
		}
	}
}
