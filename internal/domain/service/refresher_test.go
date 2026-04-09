package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/testutils"
)

// TestRunCancelledNoLoop тестирует, что Worker.Run корректно завершает работу
// при отмене контекста до входа в основной цикл.
//
// Важно: тест дожидается начала выполнения refresh-функции,
// чтобы избежать гонки между запуском воркера и отменой контекста.
// Без этой синхронизации возможна ситуация, когда cancel() вызывается
// до фактического старта Run, что делает поведение недетерминированным
// и может приводить к "flaky" тестам или зависаниям.
//
// Ожидаемое поведение: Run должен завершиться без ошибки (nil),
// так как отмена контекста считается штатным сценарием завершения.
func TestRunCancelledNoLoop(t *testing.T) {
	var buf bytes.Buffer
	logger := testutils.InitLogger(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})

	refreshCfg, err := NewRefreshConfig(50*time.Millisecond, time.Second, 0)
	if err != nil {
		t.Fatalf("failed to create refresh config: %v", err)
	}

	refresher := NewRefresher(refreshCancelledWithSignal(started), refreshCfg, logger)

	ch := make(chan error, 1)

	go func() {
		ch <- refresher.Run(ctx)
	}()

	// Ждём, пока refresh реально начался
	<-started
	// Теперь безопасно отменять
	cancel()

	err = <-ch
	if err != nil {
		t.Fatalf("expected nil error on cancel, got: %v", err)
	}
}

// TestRunFailedPermanentNoLoop тестирует поведение Worker.Run,
// когда первая попытка refresh завершается с постоянной (permanent) ошибкой.
//
// В этом сценарии refresh-функция сразу возвращает domain.ErrPermanent,
// что означает неустранимую ошибку, при которой дальнейшие попытки
// обновления кэша не имеют смысла.
//
// Тест синхронизируется с началом выполнения refresh через канал started,
// чтобы гарантировать, что воркер уже начал выполнение,
// и исключить гонку между запуском Run и завершением теста.
//
// Ожидаемое поведение: Worker.Run должен завершиться с ошибкой,
// обернутой вокруг domain.ErrPermanent.
func TestRunFailedPermanentNoLoop(t *testing.T) {
	var buf bytes.Buffer
	logger := testutils.InitLogger(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})

	refreshCfg, err := NewRefreshConfig(50*time.Millisecond, time.Second, 0)
	if err != nil {
		t.Fatalf("failed to create refresh config: %v", err)
	}

	refresher := NewRefresher(refreshPermanentErrWithSignal(started), refreshCfg, logger)

	ch := make(chan error, 1)

	go func() {
		ch <- refresher.Run(ctx)
	}()

	// Ждём, пока refresh реально начался
	<-started

	err = <-ch
	if !errors.Is(err, domain.ErrPermanent) {
		t.Fatalf("expected ErrPermanent error on cancel, got: %v", err)
	}
}

// TestRunFailedTransientNoLoop тестирует поведение Worker.Run,
// когда refresh-функция возвращает временную (transient) ошибку.
//
// В этом сценарии ошибка не является фатальной, поэтому воркер
// не должен завершать работу и обязан продолжить выполнение
// (перейти к следующей итерации по тикеру).
//
// Тест синхронизируется с началом выполнения refresh через канал started,
// чтобы гарантировать, что воркер уже начал обработку,
// и исключить race condition между запуском Run и проверкой его состояния.
//
// После первого вызова refresh тест проверяет, что Worker.Run
// не завершился преждевременно (что подтверждает корректную обработку transient ошибки).
// Затем контекст отменяется, и ожидается, что воркер завершится штатно,
// вернув nil.
func TestRunFailedTransientNoLoop(t *testing.T) {
	var buf bytes.Buffer
	logger := testutils.InitLogger(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})

	refreshCfg, err := NewRefreshConfig(50*time.Millisecond, time.Second, 0)
	if err != nil {
		t.Fatalf("failed to create refresh config: %v", err)
	}

	refresher := NewRefresher(refreshTransientErrWithSignalNoLoop(started), refreshCfg, logger)

	ch := make(chan error, 1)

	go func() {
		ch <- refresher.Run(ctx)
	}()

	// дождались первого refresh
	<-started

	// Проверяем, что Run НЕ завершился
	select {
	case err := <-ch:
		t.Fatalf("Run should not exit on transient error, got: %v", err)
	case <-time.After(100 * time.Millisecond):
		// ок, жив
	}
	// теперь останавливаем
	cancel()

	err = <-ch
	if err != nil {
		t.Fatalf("expected nil error on cancel, got: %v", err)
	}
}

// TestRunLoopStopsOnCtxDone проверяет, что воркер:
// 1. После первого refresh продолжает работать (не завершился преждевременно)
// 2. Корректно выходит из цикла при отмене контекста (ctx.Done)
func TestRunLoopStopsOnCtxDone(t *testing.T) {
	var buf bytes.Buffer
	logger := testutils.InitLogger(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})

	refreshCfg, err := NewRefreshConfig(50*time.Millisecond, time.Second, 0)
	if err != nil {
		t.Fatalf("failed to create refresh config: %v", err)
	}

	refresher := NewRefresher(refreshCancelledWithSignalLoop(started), refreshCfg, logger)

	ch := make(chan error, 1)

	go func() {
		ch <- refresher.Run(ctx)
	}()
	// дождались первого refresh
	<-started

	// Проверяем, что Run НЕ завершился
	select {
	case err := <-ch:
		t.Fatalf("expected nil error on the tick, got: %v", err)
	case <-time.After(100 * time.Millisecond):
		// ок, жив
	}

	// теперь останавливаем
	cancel()

	err = <-ch
	if err != nil {
		t.Fatalf("expected nil error on cancel, got: %v", err)
	}
}

// TestRunFailedPermanentLoop тестирует поведение Run, когда мы обновляем кэш в бесконечном цикле,
// когда мы получили permanent ошибку
func TestRunFailedPermanentLoop(t *testing.T) {
	var buf bytes.Buffer
	logger := testutils.InitLogger(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	refreshCfg, err := NewRefreshConfig(500*time.Millisecond, 50*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("failed to create refresh config: %v", err)
	}

	refresher := NewRefresher(
		refreshPermanentErrOnSecondCall(firstStarted, secondStarted),
		refreshCfg,
		logger,
	)

	ch := make(chan error, 1)

	go func() {
		ch <- refresher.Run(ctx)
	}()

	// Первый refresh стартовал и завершился успешно.
	<-firstStarted

	// Второй refresh стартовал уже внутри loop.
	<-secondStarted

	err = <-ch
	if !errors.Is(err, domain.ErrPermanent) {
		t.Fatalf("expected ErrPermanent, got: %v", err)
	}
}

// TestRunFailedTransientLoop тестирует поведение Run, когда мы обновляем кэш в бесконечном цикле,
// когда мы получили transient ошибку
func TestRunFailedTransientLoop(t *testing.T) {
	var buf bytes.Buffer
	logger := testutils.InitLogger(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	refreshCfg, err := NewRefreshConfig(500*time.Millisecond, 50*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("failed to create refresh config: %v", err)
	}

	refresher := NewRefresher(
		refreshTransientErrOnSecondCall(firstStarted, secondStarted),
		refreshCfg,
		logger,
	)

	ch := make(chan error, 1)

	go func() {
		ch <- refresher.Run(ctx)
	}()

	// Первый refresh стартовал и успешно завершился.
	<-firstStarted

	// Второй refresh стартовал уже внутри loop и должен вернуть transient error.
	<-secondStarted

	// После transient ошибки Run не должен завершиться.
	select {
	case err := <-ch:
		t.Fatalf("worker exited unexpectedly after transient error, err: %v", err)
	case <-time.After(100 * time.Millisecond):
		// ok, worker is still running
	}

	// Теперь останавливаем воркер штатно.
	cancel()

	err = <-ch
	if err != nil {
		t.Fatalf("expected nil error on cancel, got: %v", err)
	}
}

// refreshCancelledWithSignal возвращает тестовую реализацию refresh-функции,
// которая сигнализирует о начале выполнения через канал started.
//
// При первом входе в функцию канал started закрывается,
// что позволяет тесту синхронизироваться и точно знать,
// что выполнение refresh уже началось.
//
// Это необходимо для детерминированного тестирования:
// без такого сигнала невозможно гарантировать,
// что воркер уже начал обработку перед отменой контекста.
//
// После сигнала функция либо:
//
//   - завершится успешно через заданное время,
//
//   - либо завершится с domain.ErrCancelled при отмене контекста.
func refreshCancelledWithSignal(started chan struct{}) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// сигнал: мы внутри refresh
		close(started)

		select {
		case <-time.After(50 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return domain.ErrCancelled
		}
	}
}

func refreshCancelledWithSignalLoop(started chan struct{}) func(ctx context.Context) error {
	var once sync.Once
	return func(ctx context.Context) error {
		// сигнал: мы внутри refresh
		once.Do(func() { close(started) })
		select {
		case <-time.After(40 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return domain.ErrCancelled
		}
	}
}

// refreshPermanentErrWithSignal возвращает тестовую реализацию refresh-функции,
// которая сигнализирует о начале выполнения через канал started
// и сразу завершает работу с domain.ErrPermanent.
//
// Используется для тестирования сценария,
// когда воркер получает неустранимую (permanent) ошибку
// уже на первой попытке обновления.
func refreshPermanentErrWithSignal(started chan struct{}) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// сигнал: мы внутри refresh
		close(started)
		return fmt.Errorf("%w: %v", domain.ErrPermanent, errors.New("permanent"))
	}
}

// refreshTransientErrWithSignalNoLoop возвращает тестовую реализацию refresh-функции,
// которая сигнализирует о начале выполнения через канал started
// и возвращает временную (transient) ошибку.
//
// Сигнал отправляется только один раз с помощью sync.Once,
// так как refresh может вызываться многократно в рамках цикла Worker.Run.
// Это предотвращает panic при повторном закрытии канала.
//
// Используется для тестирования сценариев,
// в которых воркер должен продолжать выполнение после transient ошибки,
// а не завершаться.
func refreshTransientErrWithSignalNoLoop(started chan struct{}) func(ctx context.Context) error {
	var once sync.Once
	return func(ctx context.Context) error {
		once.Do(func() {
			close(started)
		})
		return io.ErrUnexpectedEOF

	}
}

// refreshTransientErrOnSecondCall возвращает тестовую refresh-функцию,
// которая:
//
//   - на первом вызове сигнализирует через firstStarted и возвращает nil
//   - на втором вызове сигнализирует через secondStarted и возвращает transient error
//   - на последующих вызовах возвращает nil
//
// Используется для тестирования сценария, в котором transient ошибка
// происходит именно внутри loop после первого успешного refresh.
func refreshTransientErrOnSecondCall(firstStarted, secondStarted chan struct{}) func(ctx context.Context) error {
	callNum := 0

	return func(ctx context.Context) error {
		callNum++
		currentCall := callNum

		switch currentCall {
		case 1:
			close(firstStarted)
			return nil
		case 2:
			close(secondStarted)
			return io.ErrUnexpectedEOF
		default:
			return nil
		}
	}
}

// refreshPermanentErrOnSecondCall возвращает тестовую refresh-функцию,
// которая:
//   - на первом вызове сигнализирует через firstStarted и возвращает nil
//   - на втором вызове сигнализирует через secondStarted и возвращает domain.ErrPermanent
//   - на последующих вызовах возвращает nil
//
// Используется для тестирования сценария, в котором permanent ошибка
// происходит внутри loop после первого успешного refresh.
func refreshPermanentErrOnSecondCall(firstStarted, secondStarted chan struct{}) func(ctx context.Context) error {
	callNum := 0

	return func(ctx context.Context) error {
		callNum++

		switch callNum {
		case 1:
			close(firstStarted)
			return nil
		case 2:
			close(secondStarted)
			return fmt.Errorf("%w", domain.ErrPermanent)
		default:
			return nil
		}
	}
}
