package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with sample data",
	Long:  `Seed the database with sample data for development and testing purposes.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig(".")
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		db, err := initDB(cfg.Database)
		if err != nil {
			log.Fatalf("failed to init db: %v", err)
		}

		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		fadhilEmail := "fadhil@mail.com"
		fadhilName := "Fadhil"
		var exists int
		row := db.Raw("SELECT 1 FROM users WHERE email = ?", fadhilEmail).Row()
		fadhilExists := false
		if err := row.Scan(&exists); err == nil {
			fmt.Println("fadhil user already exists; will ensure permissions")
			fadhilExists = true
		}

		if !fadhilExists {
			if err := db.Exec("INSERT INTO users (email, name, password_hash, is_active, created_at, updated_at) VALUES (?, ?, ?, true, now(), now())", fadhilEmail, fadhilName, string(hash)).Error; err != nil {
				log.Fatalf("failed to insert fadhil user: %v", err)
			}
			fmt.Println("Seeded fadhil user:", fadhilEmail)
		}

		adminEmail := "padil@mail.com"
		adminName := "Padil Admin"
		row = db.Raw("SELECT 1 FROM users WHERE email = ?", adminEmail).Row()
		adminExists := false
		if err := row.Scan(&exists); err == nil {
			fmt.Println("admin user already exists; will ensure permissions")
			adminExists = true
		}

		if !adminExists {
			if err := db.Exec("INSERT INTO users (email, name, password_hash, is_active, created_at, updated_at) VALUES (?, ?, ?, true, now(), now())", adminEmail, adminName, string(hash)).Error; err != nil {
				log.Fatalf("failed to insert admin user: %v", err)
			}
			fmt.Println("Seeded admin user:", adminEmail)
		}

		permissions := []struct {
			Name string
			Desc string
		}{
			{"admin", "full administrator"},
			{"approve_expenses", "Can approve expenses"},
			{"view_expenses", "Can view expenses"},
			{"reject_expenses", "Can reject expenses"},
			{"create_expenses", "Can create expenses"},
			{"edit_expenses", "Can edit expenses"},
			{"retry_payments", "Can retry payments"},
		}

		for _, p := range permissions {
			var pid int64
			row := db.Raw("SELECT id FROM permissions WHERE name = ?", p.Name).Row()
			if err := row.Scan(&pid); err != nil {

				if err := db.Exec("INSERT INTO permissions (name, description, created_at) VALUES (?, ?, now())", p.Name, p.Desc).Error; err != nil {
					log.Fatalf("failed to insert permission %s: %v", p.Name, err)
				}
			}
		}

		var adminUserID int64
		if err := db.Raw("SELECT id FROM users WHERE email = ?", adminEmail).Row().Scan(&adminUserID); err != nil {
			log.Fatalf("failed to lookup admin user id: %v", err)
		}

		for _, p := range permissions {
			var pid int64
			if err := db.Raw("SELECT id FROM permissions WHERE name = ?", p.Name).Row().Scan(&pid); err != nil {
				log.Fatalf("permission not found after insert %s: %v", p.Name, err)
			}

			var exists int
			if err := db.Raw("SELECT 1 FROM user_permissions WHERE user_id = ? AND permission_id = ?", adminUserID, pid).Row().Scan(&exists); err == nil {
				continue
			}

			if err := db.Exec("INSERT INTO user_permissions (user_id, permission_id, granted_by, created_at) VALUES (?, ?, NULL, now())", adminUserID, pid).Error; err != nil {
				log.Fatalf("failed to grant permission %s to admin user: %v", p.Name, err)
			}
		}

		fmt.Println("Granted all permissions to admin user:", adminEmail)

		var fadhilUserID int64
		if err := db.Raw("SELECT id FROM users WHERE email = ?", fadhilEmail).Row().Scan(&fadhilUserID); err != nil {
			log.Fatalf("failed to lookup fadhil user id: %v", err)
		}

		fadhilUserPermissions := []string{"view_expenses", "create_expenses"}
		for _, permName := range fadhilUserPermissions {
			var pid int64
			if err := db.Raw("SELECT id FROM permissions WHERE name = ?", permName).Row().Scan(&pid); err != nil {
				log.Fatalf("permission not found %s: %v", permName, err)
			}

			var exists int
			if err := db.Raw("SELECT 1 FROM user_permissions WHERE user_id = ? AND permission_id = ?", fadhilUserID, pid).Row().Scan(&exists); err == nil {
				continue
			}

			if err := db.Exec("INSERT INTO user_permissions (user_id, permission_id, granted_by, created_at) VALUES (?, ?, NULL, now())", fadhilUserID, pid).Error; err != nil {
				log.Fatalf("failed to grant permission %s to fadhil user: %v", permName, err)
			}
		}

		fmt.Println("Granted limited permissions to fadhil user (can only create expenses):", fadhilEmail)

		categories := []struct {
			Name string
			Desc string
		}{
			{"perjalanan", "perjalanan dinas dan transportasi"},
			{"makan", "makan dan hiburan"},
			{"kantor", "perlengkapan, peralatan kantor"},
			{"liburan", "biaya liburan dan rekreasi"},
			{"lain_lain", "biaya lain-lain"},
		}

		for _, c := range categories {
			var exists int
			row := db.Raw("SELECT 1 FROM expense_categories WHERE name = ?", c.Name).Row()
			if err := row.Scan(&exists); err != nil {

				if err := db.Exec("INSERT INTO expense_categories (name, description, is_active, created_at) VALUES (?, ?, true, now())", c.Name, c.Desc).Error; err != nil {
					log.Fatalf("failed to insert expense category %s: %v", c.Name, err)
				}
				fmt.Printf("Seeded expense category: %s\n", c.Name)
			}
		}

		fmt.Println("Expense categories seeded successfully")
	},
}
