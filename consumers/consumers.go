package consumers

import (
	"github.com/bitly/go-nsq"
	"github.com/kobeld/qortex-realtime/configs"
	"github.com/kobeld/qortex-realtime/consumers/nfts"
	"github.com/theplant/qortex/utils"
)

type Consumer interface {
	TopicAndChannel() (string, string)
	HandleMessage(*nsq.Message) error
}

func InitConsumers() (err error) {

	// Put consumers here
	consumers := []Consumer{
		&nfts.EntryNtfsConsumer{},
	}

	for _, consumer := range consumers {

		topic, channel := consumer.TopicAndChannel()
		reader, err := nsq.NewReader(topic, channel)
		if err != nil {
			utils.PrintStackAndError(err)
			return err
		}

		reader.AddHandler(consumer)

		err = reader.ConnectToLookupd(configs.NsqLookupAdddr)
		if err != nil {
			utils.PrintStackAndError(err)
			return err
		}
	}

	return
}
