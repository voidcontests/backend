package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/lib/logger/sl"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
	"github.com/xssnick/tonutils-go/address"
)

// TODO: Wrap errors

type TonProofService struct {
	repo    *repository.Repository
	ton     *ton.Client
	testnet bool
}

func NewTonProofService(repo *repository.Repository, tc *ton.Client) *TonProofService {
	return &TonProofService{
		repo:    repo,
		ton:     tc,
		testnet: tc.IsTestnet(),
	}
}

func (s *TonProofService) GeneratePayload() (string, error) {
	op := "service.TonProofService.GeneratePayload"

	payload, err := s.ton.TonConnect.GeneratePayload()
	if err != nil {
		slog.Error("tonproof: failed to generate payload", slog.String("op", op), slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("tonproof: payload generated", slog.String("payload", payload))
	return payload, nil
}

func (s *TonProofService) VerifyProofAndSetAddress(ctx context.Context, userID int, tp request.TonProof) error {
	op := "service.TonProofService.VerifyProofAndSetAddress"

	expectedNetwork := ton.MainnetID
	if s.testnet {
		expectedNetwork = ton.TestnetID
	}
	if tp.Network != expectedNetwork {
		return fmt.Errorf("%s: network mismatch", op)
	}

	proof := tonconnect.Proof{
		Address: tp.Address,
		Proof: tonconnect.ProofData{
			Timestamp: tp.Proof.Timestamp,
			Domain:    tp.Proof.Domain.Value,
			Signature: tp.Proof.Signature,
			Payload:   tp.Proof.Payload,
			StateInit: tp.Proof.StateInit,
		},
	}

	allowAnyDomain := func(addr string) (bool, error) {
		return true, nil
	}

	verified, _, err := s.ton.TonConnect.CheckProof(ctx, &proof, s.ton.TonConnect.CheckPayload, allowAnyDomain)
	if err != nil {
		slog.Error("tonproof: proof verification failed", slog.String("op", op), sl.Err(err))
		return ErrTonProofFailed
	}
	if !verified {
		slog.Warn("tonproof: proof not verified", slog.String("op", op))
		return ErrTonProofFailed
	}

	addr, err := address.ParseRawAddr(tp.Address)
	if err != nil {
		slog.Warn("tonproof: failed to parse raw address", sl.Err(err))
		return fmt.Errorf("%s: failed to parse address: %w", op, err)
	}

	addrStr := addr.Testnet(s.testnet).String()

	_, err = s.repo.User.UpdateUser(ctx, userID, models.UpdateUserParams{
		Address: &addrStr,
	})
	if err != nil {
		return fmt.Errorf("%s: failed to update user: %w", op, err)
	}

	return nil
}
