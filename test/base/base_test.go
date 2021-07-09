// +build e2e

package base

import (
	"github.com/Shopify/sarama"
	"github.com/strimzi/strimzi-canary/test"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)


const (
	httpUrlPrefix                   = "http://localhost:8080"
	metricsEndpoint                 = "/metrics"
	canaryTopicName                 = "__strimzi_canary"
	metricServerUpdateTimeInSeconds = 30
	kafkaMainBroker = "localhost:9092"
)

/* test checks for following:
*  the presence of canary topic,
*  liveliness of topic (messages being produced),
*/
func TestCanaryTopicLiveliness(t *testing.T) {
	log.Println("TestCanaryTopic test starts")

	// setting up timeout
	timeout := time.After(40 * time.Second)
	testDone := make(chan bool)

	// test itself.
	go func() {

		config := sarama.NewConfig()
		config.Consumer.Return.Errors = true

		//kafka end point
		brokers := []string{kafkaMainBroker}

		//get broker
		cluster, err := sarama.NewConsumer(brokers, config)
		if err != nil {
			t.Error(err.Error())
		}

		// get all topics
		topics, _ := cluster.Topics()
		if !test.IsTopicPresent(canaryTopicName, topics) {
			t.Errorf("%s is not present", canaryTopicName)
		}

		// consume single message
		consumer, _ := sarama.NewConsumer(brokers, nil)
		partitionConsumer, _ := consumer.ConsumePartition(canaryTopicName, 0, 0)
		msg := <-partitionConsumer.Messages()
		log.Printf("Consumed: offset:%d  value:%v", msg.Offset, string(msg.Value))
		testDone <- true

	}()

	select {
	case <-timeout:
		t.Error("Test didn't finish in time due to message not being read in time")
	case <-testDone:
		log.Println("message received")
	}

}

func TestEndpointsAvailability(t *testing.T) {
	log.Println("TestEndpointsAvailability test starts")
	var testInputs = [...]struct{
		endpoint           string
		expectedStatusCode int
	}{
		{metricsEndpoint, 200},
		{"/liveness", 200},
		{"/readiness", 200},
		{"/invalid", 404},
	}

	for _, testInput := range testInputs {
		var completeUrl = httpUrlPrefix + testInput.endpoint
		resp, err := http.Get(completeUrl)
		if err != nil {
			t.Errorf("Http server unreachable for url: %s",completeUrl  )
		}

		wantResponseStatus := testInput.expectedStatusCode
		gotResponseStatus := resp.StatusCode
		if wantResponseStatus != gotResponseStatus {
			t.Errorf("endpoint: %s expected response code: %d obtained: %d" ,completeUrl,wantResponseStatus,gotResponseStatus  )
		}
		log.Printf("endpoint:  %s, responded with expected status code %d\n", testInput.endpoint, testInput.expectedStatusCode)
	}
}

func TestMetricServerContentUpdating(t *testing.T) {
	log.Println("TestMetricServerContentUpdating test starts")

	resp, _ := http.Get(httpUrlPrefix + metricsEndpoint)
	body, _ := ioutil.ReadAll(resp.Body)
	totalRequestCountT1 := test.ParseCountFromMetrics(string(body))
	if len(totalRequestCountT1) < 1 {
		t.Errorf("Content of metric server is not updated as expected")
	}

	// test  has to wait for Defined time before next round of data producing is finished.
	time.Sleep(time.Second * (metricServerUpdateTimeInSeconds + 2))
	resp, _ = http.Get(httpUrlPrefix + metricsEndpoint)
	body, _ = ioutil.ReadAll(resp.Body)

	// totalRequestCountT2 stores value produced after defined number of seconds from obtaining totalRequestCountT1
	totalRequestCountT2 := test.ParseCountFromMetrics(string(body))
	if totalRequestCountT2 <= totalRequestCountT1{
		t.Errorf("Data are not updated within requested time period %d on endpoint %s", metricServerUpdateTimeInSeconds, metricsEndpoint)
	}

}




