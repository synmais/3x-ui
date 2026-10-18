package job

import "github.com/mhsanaei/3x-ui/v3/internal/web/service/tgbot"

// SynVPNNotifyJob checks traffic and expiry thresholds for linked subscribers.
type SynVPNNotifyJob struct {
	bot tgbot.Tgbot
}

func NewSynVPNNotifyJob() *SynVPNNotifyJob { return new(SynVPNNotifyJob) }

func (j *SynVPNNotifyJob) Run() { j.bot.NotifySynVPNUsers() }
