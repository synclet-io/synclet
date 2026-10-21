package authservice

import (
	strconv "strconv"
	// user code 'imports'
	// end user code 'imports'
)

type OAuthAccessLevel byte

const (
	OAuthAccessLevelRead OAuthAccessLevel = iota + 1
	OAuthAccessLevelActions
	OAuthAccessLevelWrite
)

// user code 'OAuthAccessLevel methods'
// end user code 'OAuthAccessLevel methods'
func (o OAuthAccessLevel) IsValid() bool {
	return o > 0 && o < 4
}
func (o OAuthAccessLevel) IsRead() bool {
	return o == OAuthAccessLevelRead
}
func (o OAuthAccessLevel) IsActions() bool {
	return o == OAuthAccessLevelActions
}
func (o OAuthAccessLevel) IsWrite() bool {
	return o == OAuthAccessLevelWrite
}
func (o OAuthAccessLevel) String() string {
	const names = "ReadActionsWrite"

	var indexes = [...]int32{0, 4, 11, 16}
	if o < 1 || o > 3 {
		return "OAuthAccessLevel(" + strconv.FormatInt(int64(o), 10) + ")"
	}

	return names[indexes[o-1]:indexes[o]]
}
