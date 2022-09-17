package lru

import "container/list"

type Cache struct {
	maxBytes  int64 // 允许使用的最大内存
	nbytes    int64 // 当前已使用的内存
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value) // 记录被移除时的回调函数
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int64
}

func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		OnEvicted: onEvicted,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
	}
}

// Get lookup a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		e := element.Value.(*entry)
		return e.value, ok
	}
	return
}

// RemoveOldest remove the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		e := ele.Value.(*entry)
		delete(c.cache, e.key)
		c.nbytes -= int64(len(e.key)) + int64(e.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(e.key, e.value)
		}
	}
}

// Add a value to the cache
func (c *Cache) Add(key string, value Value) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		e := element.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(e.value.Len())
		e.value = value
	} else {
		element := c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = element
		c.nbytes += int64(len(key)) + value.Len()
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
