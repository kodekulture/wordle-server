package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/lordvidex/errs/v2"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/repository"
	"github.com/kodekulture/wordle-server/service/hasher"
)

type gameService struct {
	gr repository.Game
	pr repository.Player
	h  hasher.Bcrypt
}

func (s *gameService) CreatePlayer(ctx context.Context, player *game.Player) error {
	if player == nil {
		return ErrNoPlayer
	}
	var err error
	player.Password, err = s.h.Hash(player.Password)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "password processing error")
	}
	err = s.pr.Create(ctx, *player)
	if err != nil {
		return errs.WrapCode(err, errs.InvalidArgument, "error creating player")
	}
	return nil
}

func (s *gameService) GetPlayer(ctx context.Context, username string) (*game.Player, error) {
	p, err := s.pr.GetByUsername(ctx, username)
	if err != nil {
		return nil, errs.WrapCode(err, errs.NotFound, "player not found")
	}
	return p, nil
}

func (s *gameService) UpdatePlayerSession(ctx context.Context, username string, sessionTs int64) error {
	return s.pr.UpdatePlayerSession(ctx, username, sessionTs)
}

func (s *gameService) ComparePasswords(hash, original string) error {
	return s.h.Compare(hash, original)
}

func (s *gameService) GetGame(ctx context.Context, userID int, roomID uuid.UUID) (*game.Game, error) {
	room, err := s.gr.FetchGame(ctx, userID, roomID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "error fetching game")
	}
	return room, nil
}

func (s *gameService) GetPlayerRooms(ctx context.Context, playerID int) ([]game.Game, error) {
	rooms, err := s.gr.GetGames(ctx, playerID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "error fetching games")
	}
	return rooms, nil
}

func (s *gameService) StartGame(ctx context.Context, g *game.Game) error {
	err := s.gr.StartGame(ctx, g)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "error saving game for all players")
	}
	return nil
}

func (s *gameService) WipeGameData(ctx context.Context, id uuid.UUID) error {
	err := s.gr.WipeGameData(ctx, id)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "error saving game for all players")
	}
	return nil
}

func (s *gameService) FinishGame(ctx context.Context, g *game.Game) error {
	err := s.gr.FinishGame(ctx, g)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "error saving game for all players")
	}
	return nil
}

func newGameSrv(gr repository.Game, pr repository.Player) *gameService {
	return &gameService{gr, pr, hasher.Bcrypt{}}
}
