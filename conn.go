package pool

import (
	"net"
	"sync"
)

type PoolConn struct {
	net.Conn
	mux      sync.RWMutex
	pool     *channelPool
	unusable bool
}

func (this *PoolConn) Close() error {
	this.mux.RLock()
	defer this.mux.RUnlock()

	if this.unusable {
		if this.Conn != nil {
			return this.Conn.Close()
		}
		return nil
	}
	return this.pool.put(this.Conn)
}

func (this *PoolConn) MarkUnusable() {
	this.mux.Lock()
	this.unusable = true
	this.mux.Unlock()
}

func (this *channelPool) wrapConn(conn net.Conn) net.Conn {
	p := &PoolConn{pool: this}
	p.Conn = conn
	return p
}
