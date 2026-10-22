package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type App struct {
	db       *gorm.DB
	log       *slog.Logger
	turnstileSecret string
	mailer          *mailer
	limiter         *rateLimiter
	notifications   sync.WaitGroup
}

const maxFormBytes = 64 << 10

// maxInquiriesPerDay caps how many inquiries are saved in any rolling 24
// hours, so a flood that gets past Turnstile can't fill the database.
const maxInquiriesPerDay = 50

func (a *App) createInquiry(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.limiter.allow(ip, time.Now()) {
		a.log.Warn("rate limited", "ip", ip)
		http.Error(w, "Too many submissions. Please try again later.", http.StatusTooManyRequests)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission.", http.StatusBadRequest)
		return
	}

	if r.PostFormValue("website") != "" {
		http.Redirect(w, r, "/thanks/", http.StatusSeeOther)
		return
	}

	saved, err := a.recentInquiries(r.Context(), time.Now())
	if err != nil {
		a.log.Error("count recent inquiries", "err", err)
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}
	if saved >= maxInquiriesPerDay {
		a.log.Warn("daily inquiry cap reached", "count", saved)
		http.Error(w, "We've received an unusually high number of requests today. Please try again tomorrow.", http.StatusServiceUnavailable)
		return
	}

	ok, err := verifyTurnstile(r.Context(), a.turnstileSecret, r.PostFormValue("cf-turnstile-response"), r.Header.Get("CF-Connecting-IP"))
	if err != nil {
		a.log.Error("turnstile verify", "err", err)
		http.Error(w, "Verification is temporarily unavailable. Please try again.", http.StatusServiceUnavailable)
		return
	}
	if !ok {
		http.Error(w, "Verification failed. Please go back and try again.", http.StatusBadRequest)
		return
	}

	c := Inquiry{
		Name:  r.PostFormValue("name"),
		Email:  r.PostFormValue("email"),
		Phone:  r.PostFormValue("phone"),
		Street: r.PostFormValue("street"),
		City:  r.PostFormValue("city"),
		State:  r.PostFormValue("state"),
		Zip:   r.PostFormValue("zip"),
		Message: r.PostFormValue("message"),
	}
	if err := validate(&c); err != nil {
		http.Error(w, "Please go back and check your name, email, ZIP, and project details.", http.StatusBadRequest)
		return
	}

	err = a.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		customer := Customer{Email: c.Email}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&customer).Error; err != nil {
			return err
		}
		return tx.Create(&c).Error
	})
	if err != nil {
		a.log.Error("insert inquiry", "err", err)
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	a.log.Info("inquiry created", "id", c.ID)
	a.notify(r.Context(), &c)
	http.Redirect(w, r, "/thanks/", http.StatusSeeOther)
}

// notify emails the inquiry unless the daily email budget is used up. The
// budget is counted from the inquiries table, so it survives restarts.
func (a *App) notify(ctx context.Context, c *Inquiry) {
	if a.mailer == nil {
		return
	}
	recent, err := a.recentInquiries(ctx, c.CreatedAt)
	if err != nil {
		a.log.Error("count recent inquiries", "err", err)
	}
	if recent > maxEmailsPerDay {
		a.log.Warn("email budget used up; not emailing inquiry", "id", c.ID)
		return
	}
	last := recent == maxEmailsPerDay
	a.notifications.Go(func() {
		if err := a.mailer.sendInquiry(c, last); err != nil {
			a.log.Error("send inquiry email", "id", c.ID, "err", err)
		}
	})
}

// recentInquiries counts inquiries saved in the 24 hours before now.
func (a *App) recentInquiries(ctx context.Context, now time.Time) (int64, error) {
	var n int64
	err := a.db.WithContext(ctx).Model(&Inquiry{}).
		Where("created_at > ?", now.Add(-24*time.Hour)).
		Count(&n).Error
	return n, err
}

func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := a.db.DB()
	if err != nil || sqlDB.PingContext(r.Context()) != nil {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}
