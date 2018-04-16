package util

//Semaphore Semaphore
type Semaphore chan struct{}

//MakeSemaphore MakeSemaphore
func MakeSemaphore(n int) Semaphore {
	return make(Semaphore, n)
}

//Acquire Acquire
func (s Semaphore) Acquire() {
	s <- struct{}{}
}

//Release Release
func (s Semaphore) Release() {
	<-s
}
