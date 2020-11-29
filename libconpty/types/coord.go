package types

type COORD struct {
	X int16
	Y int16
}

func (c *COORD) Pack() uintptr {
	return uintptr((int32(c.Y) << 16) | int32(c.X))
}
