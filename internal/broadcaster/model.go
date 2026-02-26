package broadcaster

import "sync"

type Broadcaster struct {
	clients    map[chan string]bool
	mu         sync.Mutex
	Register   chan chan string
	Unregister chan chan string
	Messages   chan string
}

func New() *Broadcaster {
	b := &Broadcaster{
		clients:    make(map[chan string]bool),
		Register:   make(chan chan string),
		Unregister: make(chan chan string),
		Messages:   make(chan string, 100),
	}
	go b.run()
	return b
}

func (b *Broadcaster) run() {
	for {
		select {
		case client := <-b.Register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
		case client := <-b.Unregister:
			b.mu.Lock()
			delete(b.clients, client)
			close(client)
			b.mu.Unlock()
		case message := <-b.Messages:
			b.mu.Lock()
			for client := range b.clients {
				select {
				case client <- message:
				default:
					close(client)
					delete(b.clients, client)
				}
			}
			b.mu.Unlock()
		}
	}
}
