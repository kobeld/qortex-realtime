package services

import (
	"github.com/theplant/qortex/utils"
)

type Pulse int

type PulseInput struct{}

func (this *Pulse) Send(input *PulseInput, reply *string) (err error) {

	defer func() {
		if e := recover(); e != nil {
			utils.PrintStackAndError(e.(error))
		}
	}()
	*reply = "Pulse.Get"
	return
}
