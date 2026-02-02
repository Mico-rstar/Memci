package segment

import "memci/message"

type Segment struct {
	sysMsgs		[]message.Message
	UsrMsgs		[]message.Message
}