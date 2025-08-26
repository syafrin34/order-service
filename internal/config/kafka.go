// Package config
package config

import "github.com/segmentio/kafka-go"

var KafkaBrokerUrls = []string{"localhost:9092", "localhost:9093", "localhost:9094"}

func NewKafkaWrite(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(KafkaBrokerUrls...),
		Topic:                  topic,
		Balancer:               &kafka.CRC32Balancer{},
		AllowAutoTopicCreation: true,
	}
}
