package child

import "example.org/sh1-fixture/internal/session"

func Allowed() bool { return session.Accept("member") }
