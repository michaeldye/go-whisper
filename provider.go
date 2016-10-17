package whisper

import (
	"fmt"
	"net/url"
)

const T_CONFIGURE = "CONFIGURE"
const T_MICROPAYMENT = "MICROPAYMENT"

// message sent to provider from contract owner
type Announce struct {
	AgreementId    string
	ConfigureNonce string
}

type WhisperProviderMsg struct {
	Type string `json:"type"`
}

// message sent to contract owner from provider
type Configure struct {
	// embedded
	WhisperProviderMsg
	ConfigureNonce      string            `json:"configure_nonce"`
	TorrentURL          url.URL           `json:"torrent_url"`
	ImageHashes         map[string]string `json:"image_hashes"`
	ImageSignatures     map[string]string `json:"image_signatures"` // cryptographic signatures per-image
	Deployment          string            `json:"deployment"`       // JSON docker-compose like
	DeploymentSignature string            `json:"deployment_signature"`
	DeploymentUserInfo  string            `json:"deployment_user_info"`
}

func (c Configure) String() string {
	return fmt.Sprintf("Type: %v, ConfigureNonce: %v, TorrentURL: %v, ImageHashes: %v, ImageSignatures: %v, Deployment: %v, DeploymentSignature: %v, DeploymentUserInfo: %v", c.Type, c.ConfigureNonce, c.TorrentURL.String(), c.ImageHashes, c.ImageSignatures, c.Deployment, c.DeploymentSignature, c.DeploymentUserInfo)
}

func NewConfigure(configureNonce string, torrentURL url.URL, imageHashes map[string]string, imageSignatures map[string]string, deployment string, deploymentSignature string, deploymentUserInfo string) *Configure {
	return &Configure{
		WhisperProviderMsg:  WhisperProviderMsg{Type: T_CONFIGURE},
		ConfigureNonce:      configureNonce,
		TorrentURL:          torrentURL,
		ImageHashes:         imageHashes,
		ImageSignatures:     imageSignatures,
		Deployment:          deployment,
		DeploymentSignature: deploymentSignature,
		DeploymentUserInfo:  deploymentUserInfo,
	}

}

// message sent from provider to contract owner
type Micropayment struct {
	WhisperProviderMsg
	ContractAddress string
	AgreementId     string
	Amount          uint64
	AmountToDate    uint64
	Recorded        int64
}

func (m Micropayment) String() string {
	return fmt.Sprintf("Type: %v, ContractAddress: %v, AgreementId: %v, Amount: %v, AmountToDate: %v, Recorded: %v", m.Type, m.ContractAddress, m.AgreementId, m.Amount, m.AmountToDate, m.Recorded)
}

func NewMicropayment(contractAddress string, agreementId string, amount uint64, amountToDate uint64, recorded int64) *Micropayment {
	return &Micropayment{
		WhisperProviderMsg: WhisperProviderMsg{Type: T_MICROPAYMENT},
		ContractAddress:    contractAddress,
		AgreementId:        agreementId,
		Amount:             amount,
		AmountToDate:       amountToDate,
		Recorded:           recorded,
	}
}
