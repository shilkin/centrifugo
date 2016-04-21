package libcentrifugo

type PresenceBody struct {
	Channel Channel               `json:"channel"`
	Data    map[ConnID]ClientInfo `json:"data"`
}

type HistoryBody struct {
	Channel Channel   `json:"channel"`
	Data    []Message `json:"data"`
}

type AdminMessageBody struct {
	Project ProjectKey `json:"project"`
	Message Message    `json:"message"`
}

type JoinLeaveBody struct {
	Channel Channel    `json:"channel"`
	Data    ClientInfo `json:"data"`
}

type ConnectBody struct {
	Client  *ConnID `json:"client"`
	Expired bool    `json:"expired"`
	TTL     *int64  `json:"ttl"`
}

type RefreshBody struct {
	TTL *int64 `json:"ttl"`
}

type SubscribeBody struct {
	Channel Channel `json:"channel"`
}

type UnsubscribeBody struct {
	Channel Channel `json:"channel"`
}

type PublishBody struct {
	Channel Channel `json:"channel"`
	Status  bool    `json:"status"`
}
