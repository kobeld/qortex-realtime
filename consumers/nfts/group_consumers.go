package nfts

import (
	"encoding/json"
	"github.com/bitly/go-nsq"
	"github.com/kobeld/qortex-realtime/services"
	"github.com/theplant/qortex/nsqproducers"
	"github.com/theplant/qortex/utils"
)

type GroupNtfsConsumer struct{}

func (this *GroupNtfsConsumer) TopicAndChannel() (topic, channel string) {
	return nsqproducers.GROUP_TOPIC_NAME, NFTS_CHANNEL_NAME
}

func (this *GroupNtfsConsumer) HandleMessage(msg *nsq.Message) (err error) {

	groupTopicData := new(nsqproducers.GroupTopicData)
	err = json.Unmarshal(msg.Body, &groupTopicData)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Not for sharing notification
	if groupTopicData.Status != nsqproducers.TOPIC_STATUS_SHARE {
		return
	}

	services.PushGroupNotification(groupTopicData)
	return
}
