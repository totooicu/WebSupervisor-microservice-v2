package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	client, err := setupRedisClient()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	command := os.Args[1]

	switch command {
	case "ls":
		handleLsCommand(client)
	case "add":
		handleAddCommand(client)
	case "del":
		handleDelCommand(client)
	case "help":
		printHelp()
	default:
		log.Printf("Unknown command: %s", command)
		printHelp()
	}
}

func setupRedisClient() (*redis.Client, error) {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       redisDB,
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func handleLsCommand(client *redis.Client) {
	if len(os.Args) == 2 {
		listAllStreams(client)
	} else if len(os.Args) == 3 {
		streamName := os.Args[2]
		listStreamContent(client, streamName)
	} else {
		log.Println("Usage: ls [StreamsName]")
	}
}

func handleAddCommand(client *redis.Client) {
	if len(os.Args) < 4 {
		log.Println("Usage: add <StreamsName> <content>")
		log.Println("Usage: add <StreamsName> -f <FileName>")
		return
	}

	streamName := os.Args[2]
	if os.Args[3] == "-f" {
		if len(os.Args) != 5 {
			log.Println("Usage: add <StreamsName> -f <FileName>")
			return
		}
		fileName := os.Args[4]
		addMessageFromFile(client, streamName, fileName)
	} else {
		content := os.Args[3]
		addMessage(client, streamName, content)
	}
}

func handleDelCommand(client *redis.Client) {
	if len(os.Args) < 3 {
		log.Println("Usage: del <StreamsName> [id]")
		return
	}

	streamName := os.Args[2]
	if len(os.Args) == 3 {
		deleteStream(client, streamName)
	} else if len(os.Args) == 4 {
		messageID := os.Args[3]
		deleteMessage(client, streamName, messageID)
	} else {
		log.Println("Usage: del <StreamsName> [id]")
	}
}

func listAllStreams(client *redis.Client) {
	fmt.Println("=== Redis Streams ===")
	fmt.Println()

	var cursor uint64
	foundStreams := false

	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			log.Printf("Error scanning keys: %v", err)
			return
		}

		for _, key := range keys {
			keyType, err := client.Type(ctx, key).Result()
			if err != nil {
				log.Printf("Error checking key type: %v", err)
				continue
			}

			if keyType == "stream" {
				foundStreams = true
				printStreamInfo(client, key)
				fmt.Println()
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if !foundStreams {
		fmt.Println("No streams found")
	}
}

func printStreamInfo(client *redis.Client, streamName string) {
	// Get stream length
	length, err := client.XLen(ctx, streamName).Result()
	if err != nil {
		log.Printf("Error getting stream length: %v", err)
		return
	}

	// Get last message ID
	var lastID string
	messages, err := client.XRevRangeN(ctx, streamName, "+", "-", 1).Result()
	if err != nil {
		log.Printf("Error getting last message: %v", err)
		lastID = "N/A"
	} else if len(messages) > 0 {
		lastID = messages[0].ID
	} else {
		lastID = "N/A"
	}

	fmt.Printf("Stream: %s\n", streamName)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Length: %d\n", length)
	fmt.Printf("Last ID: %s\n", lastID)
}

func listStreamContent(client *redis.Client, streamName string) {
	exists, err := client.Exists(ctx, streamName).Result()
	if err != nil {
		log.Printf("Error checking stream existence: %v", err)
		return
	}
	if exists == 0 {
		log.Printf("Stream '%s' does not exist", streamName)
		return
	}

	messages, err := client.XRevRangeN(ctx, streamName, "+", "-", 100).Result()
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		return
	}

	if len(messages) == 0 {
		fmt.Printf("No messages in stream '%s'\n", streamName)
		return
	}

	fmt.Printf("Messages in stream '%s' (showing last 100):\n", streamName)
	fmt.Println(strings.Repeat("=", 80))

	for i, msg := range messages {
		fmt.Printf("\nMessage %d:\n", i+1)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("ID: %s\n", msg.ID)
		for k, v := range msg.Values {
			fmt.Printf("  %-20s: %v\n", k, v)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Total messages: %d\n", len(messages))
}

func addMessage(client *redis.Client, streamName, content string) {
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(content), &values); err != nil {
		log.Printf("Error parsing content as JSON: %v", err)
		log.Println("Note: In PowerShell, use double quotes with escaped inner quotes:")
		log.Println(`  .\streams-manager.exe add streamName "{\"key\": \"value\"}"`)
		log.Println("Or use a here-string:")
		log.Println(`  $json = @"
  {
    "key": "value",
    "name": "test"
  }
"@`)
		log.Println(`  .\streams-manager.exe add streamName $json`)
		return
	}

	result, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: values,
	}).Result()

	if err != nil {
		log.Printf("Error adding message: %v", err)
		return
	}

	fmt.Printf("✅ Message added successfully with ID: %s\n", result)
}

func addMessageFromFile(client *redis.Client, streamName, fileName string) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Printf("Error reading file: %v", err)
		return
	}

	var values map[string]interface{}
	if err := json.Unmarshal(data, &values); err != nil {
		log.Printf("Error parsing file content as JSON: %v", err)
		return
	}

	result, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: values,
	}).Result()

	if err != nil {
		log.Printf("Error adding message: %v", err)
		return
	}

	fmt.Printf("✅ Message from file '%s' added successfully with ID: %s\n", fileName, result)
}

func deleteStream(client *redis.Client, streamName string) {
	exists, err := client.Exists(ctx, streamName).Result()
	if err != nil {
		log.Printf("Error checking stream existence: %v", err)
		return
	}
	if exists == 0 {
		log.Printf("Stream '%s' does not exist", streamName)
		return
	}

	result, err := client.Del(ctx, streamName).Result()
	if err != nil {
		log.Printf("Error deleting stream: %v", err)
		return
	}

	if result > 0 {
		fmt.Printf("✅ Stream '%s' deleted successfully\n", streamName)
	} else {
		fmt.Printf("❌ Stream '%s' does not exist\n", streamName)
	}
}

func deleteMessage(client *redis.Client, streamName, messageID string) {
	exists, err := client.Exists(ctx, streamName).Result()
	if err != nil {
		log.Printf("Error checking stream existence: %v", err)
		return
	}
	if exists == 0 {
		log.Printf("Stream '%s' does not exist", streamName)
		return
	}

	result, err := client.XDel(ctx, streamName, messageID).Result()
	if err != nil {
		log.Printf("Error deleting message: %v", err)
		return
	}

	if result > 0 {
		fmt.Printf("✅ Message with ID '%s' deleted successfully\n", messageID)
	} else {
		fmt.Printf("❌ Message with ID '%s' not found\n", messageID)
	}
}

func printHelp() {
	fmt.Println("Redis Streams Manager")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ls                    - List all streams")
	fmt.Println("  ls <StreamsName>      - List messages in specific stream")
	fmt.Println("  add <StreamsName> <content>      - Add message to stream")
	fmt.Println("  add <StreamsName> -f <FileName>  - Add message from file")
	fmt.Println("  del <StreamsName>               - Delete entire stream")
	fmt.Println("  del <StreamsName> <id>          - Delete specific message by ID")
	fmt.Println("  help                           - Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ls")
	fmt.Println("  ls mystream")
	fmt.Println("  add mystream '{\"key\": \"value\", \"name\": \"test\"}'")
	fmt.Println("  add mystream -f message.json")
	fmt.Println("  del mystream")
	fmt.Println("  del mystream 1774756676541-0")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  REDIS_HOST     - Redis host (default: localhost)")
	fmt.Println("  REDIS_PORT     - Redis port (default: 6379)")
	fmt.Println("  REDIS_PASSWORD - Redis password (optional)")
}
