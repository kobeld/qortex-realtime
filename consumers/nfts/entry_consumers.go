package nfts

import (
	"encoding/json"
	"fmt"
	"github.com/bitly/go-nsq"
	"github.com/kobeld/qortex-realtime/services"
	"github.com/theplant/qortex/nsqproducers"
	"github.com/theplant/qortex/utils"
)

const (
	NFTS_CHANNEL_NAME = "nfts"
)

type EntryNtfsConsumer struct{}

func (this *EntryNtfsConsumer) TopicAndChannel() (topic, channel string) {
	return nsqproducers.ENTRY_TOPIC_NAME, NFTS_CHANNEL_NAME
}

func (this *EntryNtfsConsumer) HandleMessage(msg *nsq.Message) (err error) {

	entryTopicData := new(nsqproducers.EntryTopicData)
	err = json.Unmarshal(msg.Body, &entryTopicData)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	services.SendEntryNotification(entryTopicData.OrgId, entryTopicData.UserId, entryTopicData.ApiEntry)

	fmt.Printf("%+v \n", entryTopicData.ApiEntry)

	return
}
