package actaboards

import (
	"encoding/json"
	"fmt"
)

type InitialWitness struct {
	ID         string
	Username   string
	PublicKey  string
	PrivateKey string
}

type blockSigningKeyFull struct {
	WifPrivKey   string `json:"wif_priv_key"`
	PubKey       string `json:"pub_key"`
	BrainPrivKey string `json:"brain_priv_key"`
}

type initialWitnessCandidate struct {
	OwnerName           string              `json:"owner_name"`
	BlockSigningKey     string              `json:"block_signing_key"`
	BlockSigningKeyFull blockSigningKeyFull `json:"block_signing_key_full"`
}

type genesisPrivateData struct {
	InitialWitnessCandidates []initialWitnessCandidate `json:"initial_witness_candidates"`
}

func GetInitialWitnesses(GenesisPrivate string) ([]InitialWitness, error) {
	var genesis genesisPrivateData

	err := json.Unmarshal([]byte(GenesisPrivate), &genesis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse genesis JSON: %w", err)
	}

	witnesses := make([]InitialWitness, len(genesis.InitialWitnessCandidates))

	for i, candidate := range genesis.InitialWitnessCandidates {
		witnesses[i] = InitialWitness{
			ID:         fmt.Sprintf("1.6.%d", i+1),
			Username:   candidate.OwnerName,
			PublicKey:  candidate.BlockSigningKeyFull.PubKey,
			PrivateKey: candidate.BlockSigningKeyFull.WifPrivKey,
		}
	}

	return witnesses, nil
}
