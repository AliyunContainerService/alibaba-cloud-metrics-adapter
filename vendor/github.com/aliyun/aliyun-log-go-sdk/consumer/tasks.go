package consumerLibrary

import (
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/go-kit/kit/log/level"
)

func (consumer *ShardConsumerWorker) consumerInitializeTask() (string, error) {
	checkpoint, err := consumer.client.getCheckPoint(consumer.shardId)
	if err != nil {
		return checkpoint, err
	}
	if checkpoint != "" && err == nil {
		consumer.consumerCheckPointTracker.setPersistentCheckPoint(checkpoint)
		return checkpoint, nil
	}

	if consumer.client.option.CursorPosition == BEGIN_CURSOR {
		cursor, err := consumer.client.getCursor(consumer.shardId, "begin")
		if err != nil {
			level.Warn(consumer.logger).Log("msg", "get beginCursor error", "shard", consumer.shardId, "error", err)
		}
		return cursor, err
	}
	if consumer.client.option.CursorPosition == END_CURSOR {
		cursor, err := consumer.client.getCursor(consumer.shardId, "end")
		if err != nil {
			level.Warn(consumer.logger).Log("msg", "get endCursor error", "shard", consumer.shardId, "error", err)
		}
		return cursor, err
	}
	if consumer.client.option.CursorPosition == SPECIAL_TIMER_CURSOR {
		cursor, err := consumer.client.getCursor(consumer.shardId, fmt.Sprintf("%v", consumer.client.option.CursorStartTime))
		if err != nil {
			level.Warn(consumer.logger).Log("msg", "get specialCursor error", "shard", consumer.shardId, "error", err)

		}
		return cursor, err
	}
	level.Info(consumer.logger).Log("msg", "CursorPosition setting error, please reset with BEGIN_CURSOR or END_CURSOR or SPECIAL_TIMER_CURSOR")
	return "", errors.New("CursorPositionError")
}

func (consumer *ShardConsumerWorker) consumerFetchTask() (*sls.LogGroupList, string, error) {
	logGroup, next_cursor, err := consumer.client.pullLogs(consumer.shardId, consumer.nextFetchCursor)
	return logGroup, next_cursor, err
}

func (consumer *ShardConsumerWorker) consumerProcessTask() string {
	// If the user's consumption function reports a panic error, it will be captured and exited.
	rollBackCheckpoint := ""
	defer func() {
		if r := recover(); r != nil {
			level.Error(consumer.logger).Log("msg", "get panic in your process function", "error", r)
		}
	}()
	if consumer.lastFetchLogGroupList != nil {
		rollBackCheckpoint = consumer.process(consumer.shardId, consumer.lastFetchLogGroupList)
		consumer.consumerCheckPointTracker.flushCheck()
	}
	return rollBackCheckpoint
}
