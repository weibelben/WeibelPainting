package main

import (
	"log/slog"
	"net/http"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type App struct {
	db       *gorm.DB
	log       *slog.Logger
	turnstileSecret string
}

const maxFormBytes = 64 << 10

func (a *App) createInquiry(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission.", http.StatusBadRequest)
		return
	}

	if r.PostFormValue("website") != "" {
		http.Redirect(w, r, "/thanks/", http.StatusSeeOther)
		return
	}

	ip := r.Header.Get("CF-Connecting-IP")
	ok, err := verifyTurnstile(r.Context(), a.turnstileSecret, r.PostFormValue("cf-turnstile-response"), ip)
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
		http.Error(w, "Please go back and check your name, email, and project details.", http.StatusBadRequest)
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
	http.Redirect(w, r, "/thanks/", http.StatusSeeOther)
}

func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := a.db.DB()
	if err != nil || sqlDB.PingContext(r.Context()) != nil {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}
