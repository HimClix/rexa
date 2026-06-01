package engine

type thread struct {
	pc       int
	captures []CaptureSlot
}

func newThread(pc int, caps []CaptureSlot) thread {
	return thread{pc: pc, captures: CopySlots(caps)}
}
