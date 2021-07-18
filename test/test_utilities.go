package test

import (
	"github.com/Shopify/sarama"
	"log"
	"regexp"
	"sync"
)


type exampleConsumerGroupHandler struct{
	// mutex for exclusive write on message handler
	mutexWritePartitionPresence *sync.Mutex
	consumingDone               chan bool
	partitionsConsumptionSlice  []bool

}

func (h exampleConsumerGroupHandler) isEveryPartitionConsumed() bool  {
	for _,value := range h.partitionsConsumptionSlice {
		if value != true {
			return false
		}
	}
	return true
}

func (exampleConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }

func (exampleConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h exampleConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.mutexWritePartitionPresence.Lock()
		// setting for current partition that at least one message has been produced.
		h.partitionsConsumptionSlice[msg.Partition] = true
		// Check that Consumer group consumed at least one message for each partition on canary topic
		log.Println("msg. arrived")
		if h.isEveryPartitionConsumed() {
			log.Println("all partitions consumed")
			h.consumingDone <- true
		}
		h.mutexWritePartitionPresence.Unlock()
		sess.MarkMessage(msg, "")

	}
	return nil
}

func parseGeneratedErrorsFromMetrics(input string) string{
	// RE to catch any sort of error
	regex, _ := regexp.Compile("(?m)^strimzi_canary.*error_total.*\\s(\\d+)$")
	data := regex.FindStringSubmatch(input)
	if len(data) >= 1 {
		return data[1]
	}
	return ""
}

// We are only interested in counter that is produced
func parseSucReqRateFromMetrics( input string) string  {
	regex, _ := regexp.Compile("(?m)^promhttp_metric_handler_requests_total.*200...(\\d+)$")
	data := regex.FindStringSubmatch(input)
	if len(data) > 1 {
		return data[1]
	}
	return ""
}

func IsTopicPresent(topicName string, topics []string  ) bool{
	topicsAsMap := map[string]struct{}{}
	for _, topic := range topics {
		topicsAsMap[topic] = struct{}{}
	}
	_, isTopicPresent := topicsAsMap[topicName]
	return isTopicPresent
}


