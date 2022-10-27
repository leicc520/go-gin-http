package locker

import "time"

const (
	lockerPrefix = "redis@l"
	lockerExpire = time.Hour * 3
)

type IFLocker interface {
	Lock() bool
	UnLock()
}
