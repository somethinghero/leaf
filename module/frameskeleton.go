package module

import (
	"time"
)

//Looper looper interface
type Looper interface {
	Update(dt time.Duration)
}

//FrameSkeleton rewrite Run of skeleton add default case
type FrameSkeleton struct {
	Skeleton
	L Looper
}

//Run Run
func (s *FrameSkeleton) Run(closeSig chan bool) {
	var dt time.Duration
	var begin time.Time
	for {
		begin = time.Now()
		select {
		case <-closeSig:
			s.commandServer.Close()
			s.server.Close()
			for !s.g.Idle() || !s.client.Idle() {
				s.g.Close()
				s.client.Close()
			}
			return
		case ri := <-s.client.ChanAsynRet:
			s.client.Cb(ri)
		case ci := <-s.server.ChanCall:
			s.server.Exec(ci)
		case ci := <-s.commandServer.ChanCall:
			s.commandServer.Exec(ci)
		case cb := <-s.g.ChanCb:
			s.g.Cb(cb)
		case t := <-s.dispatcher.ChanTimer:
			t.Cb()
		default:
			if s.L != nil {
				s.L.Update(dt)
			}
		}
		dt = time.Since(begin)
		//极限update速度1000fps，防止cpu空转
		if dt < time.Millisecond {
			time.Sleep(time.Millisecond - dt)
			dt = time.Since(begin)
		}
	}
}
