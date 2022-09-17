package cache

import (
	pb "cache/geecachepb"
	"cache/singleflight"
	"errors"
	"log"
	"sync"
)

// Getter load data for a key
type Getter interface {
	Get(key string) ([]byte, error)
}

// PeerPicker 是定位拥有特定密钥的对等体时必须实现的接口
type PeerPicker interface {
	// PickPeer 根据传入的 key 选择相应节点
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是必须由对等体实现的接口。
type PeerGetter interface {
	// Get 从对应 group 查找缓存值
	Get(in *pb.Request, out *pb.Response) error
}

// GetterFunc implement Getter with a function
type GetterFunc func(key string) ([]byte, error)

// Get implement Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	loader    *singleflight.Group[ByteView]
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
		loader: &singleflight.Group[ByteView]{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	group := groups[name]
	mu.RUnlock()
	return group
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	return g.loader.Do(key, func() (ByteView, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return ByteView{}, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	request := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(request, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, err
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// RegisterPeers 注册一个 PeerPicker 用于选择远程节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
