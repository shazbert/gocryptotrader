package sqlite3

var sqliteSchema = []string{
	`CREATE TABLE roles (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		name text NOT NULL UNIQUE,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`,

	`CREATE TABLE clients (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  		user_name text NOT NULL UNIQUE,
		password text NOT NULL,
		email text UNIQUE,
		one_time_password text,
		password_created_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_logged_in DATETIME NOT NULL,
		enabled BOOLEAN NOT NULL
	  );`,

	`CREATE TABLE client_roles (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		client_id integer NOT NULL,
		role_id integer NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY(client_id) REFERENCES clients(id),
		FOREIGN KEY(role_id) REFERENCES roles(id)
	);`,

	`CREATE TABLE exchanges (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		exchange_name text NOT NULL UNIQUE,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`,

	`CREATE TABLE keys (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		api_key text NOT NULL,
		api_secret text NOT NULL,
		exchange_id integer NOT NULL,
		expires_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		enabled BOOLEAN NOT NULL, 
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id)
	);`,

	`CREATE TABLE client_keys (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		key_id integer NOT NULL,
		client_id integer NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY(client_id) REFERENCES clients(id),
		FOREIGN KEY(key_id) REFERENCES keys(id)
	);`,

	`CREATE TABLE role_keys (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		key_id integer NOT NULL,
		role_id INT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY(role_id) REFERENCES roles(id),
		FOREIGN KEY(key_id) REFERENCES keys(id)
	);`,

	`CREATE TABLE audit_trails (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		client_id integer NOT NULL,
		change text NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(client_id) REFERENCES clients(id)
	);`,

	`CREATE TABLE client_order_history (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		order_id text NOT NULL,
		client_id integer NOT NULL,
		exchange_id int NOT NULL,
		currency_pair text NOT NULL,
		asset_type text NOT NULL,
		order_type text NOT NULL,
		amount real NOT NULL,
		rate real NOT NULL,
		fulfilled_on DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id),
		FOREIGN KEY(client_id) REFERENCES clients(id),
		UNIQUE(exchange_id, order_id)
	);`,

	`CREATE TABLE exchange_platform_trade_history (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		order_id text NOT NULL,
		exchange_id int NOT NULL,
		currency_pair text NOT NULL,
		asset_type text NOT NULL,
		order_type text NOT NULL DEFAULT "NOT SPECIFIED",
		amount real NOT NULL,
		rate real NOT NULL,
		fulfilled_on DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id),
		UNIQUE(exchange_id, order_id)
	);`,
}
