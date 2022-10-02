package service

import (
	"net"
	"time"

	"github.com/m-mizutani/gots/set"
)

type NameMap struct {
	v4Map    map[[4]byte]map[string]time.Time
	v6Map    map[[16]byte]map[string]time.Time
	cnameMap map[string]map[string]time.Time
	now      func() time.Time
}

func NewNameMap() *NameMap {
	return &NameMap{
		v4Map:    make(map[[4]byte]map[string]time.Time),
		v6Map:    make(map[[16]byte]map[string]time.Time),
		cnameMap: make(map[string]map[string]time.Time),
		now:      time.Now,
	}
}

func (x *NameMap) InsertNameWithV4(addr [4]byte, name string, ttl uint32) {
	s, ok := x.v4Map[addr]
	if !ok {
		s = make(map[string]time.Time)
		x.v4Map[addr] = s
	}
	s[name] = time.Now().Add(time.Second * time.Duration(ttl))
}

func (x *NameMap) InsertNameWithV6(addr [16]byte, name string, ttl uint32) {
	s, ok := x.v6Map[addr]
	if !ok {
		s = make(map[string]time.Time)
		x.v6Map[addr] = s
	}
	s[name] = time.Now().Add(time.Second * time.Duration(ttl))
}

func (x *NameMap) InsertCName(name, cname string, ttl uint32) {
	s, ok := x.cnameMap[cname]
	if !ok {
		s = make(map[string]time.Time)
		x.cnameMap[cname] = s
	}
	s[name] = time.Now().Add(time.Second * time.Duration(ttl))
}

func (x *NameMap) LookupNameByAddr(addr net.IP) *set.Set[string] {
	switch len(addr) {
	case 4:
		var key [4]byte
		copy(key[:], addr[:])
		return x.LookupNameByV4(key)

	case 6:
		var key [16]byte
		copy(key[:], addr[:])
		return x.LookupNameByV6(key)

	default:
		return set.New[string]()
	}
}

func (x *NameMap) LookupNameByV4(addr [4]byte) *set.Set[string] {
	now := x.now()
	res := set.New[string]()

	nameSet, ok := x.v4Map[addr]
	if !ok {
		return res
	}

	for name, expiresAt := range nameSet {
		if now.After(expiresAt) {
			delete(nameSet, name)
		} else {
			res = res.Union(x.LookupNameByCName(name))
		}
	}

	return res
}

func (x *NameMap) LookupNameByV6(addr [16]byte) *set.Set[string] {
	now := x.now()
	res := set.New[string]()

	nameSet, ok := x.v6Map[addr]
	if !ok {
		return res
	}

	for name, expiresAt := range nameSet {
		if now.After(expiresAt) {
			delete(nameSet, name)
		} else {
			res = res.Union(x.LookupNameByCName(name))
		}
	}

	return res
}

func (x *NameMap) LookupNameByCName(name string) *set.Set[string] {
	now := x.now()
	nameSet, ok := x.cnameMap[name]
	if !ok {
		return set.New(name)
	}

	res := set.New[string]()
	for name, expiresAt := range nameSet {
		if now.After(expiresAt) {
			delete(nameSet, name)
		} else {
			res = res.Union(x.LookupNameByCName(name))
		}
	}

	return res
}
