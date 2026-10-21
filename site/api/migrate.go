package main

import "gorm.io/gorm"

func migrate(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		m := tx.Migrator()

		// The original schema stored each form submission in "customers".
		// Rename it to "inquiries" (with its index and sequence names, which
		// would otherwise collide with the new customers table).
		if m.HasTable("customers") && m.HasColumn("customers", "message") && !m.HasTable("inquiries") {
			for _, stmt := range []string{
				`ALTER TABLE customers RENAME TO inquiries`,
				`ALTER INDEX IF EXISTS customers_pkey RENAME TO inquiries_pkey`,
				`ALTER INDEX IF EXISTS idx_customers_email RENAME TO idx_inquiries_email`,
				`ALTER SEQUENCE IF EXISTS customers_id_seq RENAME TO inquiries_id_seq`,
			} {
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.AutoMigrate(&Customer{}); err != nil {
			return err
		}
		// Every inquiry needs a customer row before the foreign key is added.
		if m.HasTable("inquiries") {
			err := tx.Exec(`INSERT INTO customers (email, created_at)
				SELECT email, MIN(created_at) FROM inquiries GROUP BY email
				ON CONFLICT DO NOTHING`).Error
			if err != nil {
				return err
			}
		}
		return tx.AutoMigrate(&Inquiry{})
	})
}
