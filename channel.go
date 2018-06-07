package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

type Factory func() (net.Conn, error)

type channelPool struct {
	mux     sync.RWMutex
	conns   chan net.Conn
	factory Factory
}

func NewPool(initialCap int, maxCap int, factory Factory) (Pool, error) {

	if initialCap <= 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	pool := &channelPool{
		conns:   make(chan net.Conn, maxCap),
		factory: factory,
	}

	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("Factory is not able to fill the pool: %s", err)
		}
		pool.conns <- conn
	}
	return pool, nil
}

func (this *channelPool) getConnsAndFactory() (chan net.Conn, Factory) {
	this.mux.RLock()
	conns := this.conns
	factory := this.factory
	this.mux.RUnlock()
	return conns, factory
}

func (this *channelPool) Get() (net.Conn, error) {
	conns, factory := this.getConnsAndFactory()
	if conns == nil {
		return nil, ErrClosed
	}

	select {
	case conn := <-conns:
		if conn == nil {
			return nil, ErrClosed
		}
		return this.wrapConn(conn), nil
	default:
		conn, err := factory()
		if err != nil {
			return nil, err
		}
		return this.wrapConn(conn), nil
	}
}

func (this *channelPool) put(conn net.Conn) error {
	if conn == nil {
		return errors.New("Connection is nil. Rejecting")
	}

	this.mux.RLock()
	defer this.mux.RUnlock()

	if this.conns == nil {
		return conn.Close()
	}

	select {
	case this.conns <- conn:
		return nil
	default:
		return conn.Close()
	}
}

func (this *channelPool) Close() {
	this.mux.Lock()
	conns := this.conns
	this.conns = nil
	this.factory = nil
	this.mux.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for conn := range conns {
		conn.Close()
	}
}

func (this *channelPool) Len() int {
	conns, _ := this.getConnsAndFactory()
	return len(conns)
}
