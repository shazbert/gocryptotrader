package postgres

var postgresSchema = []string{
	`CREATE TABLE access_controls (
		id SERIAL PRIMARY KEY,
		level INT NOT NULL UNIQUE,
		name TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE clients (
		id SERIAL PRIMARY KEY,
  		user_name TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		email TEXT UNIQUE,
		one_time_password TEXT,
		access_level_id INT NOT NULL,
		password_created_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		last_logged_in TIMESTAMP NOT NULL,
		enabled BOOLEAN NOT NULL,
		FOREIGN KEY(access_level_id) REFERENCES access_controls(level)
	  );`,

	`CREATE TABLE exchanges (
		id SERIAL PRIMARY KEY,
		exchange_name TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE keys (
		id SERIAL PRIMARY KEY,
		api_key TEXT NOT NULL,
		api_secret TEXT NOT NULL,
		roles TEXT NOT NULL,
		client_id INT NOT NULL,
		exchange_id INT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		enabled BOOLEAN NOT NULL, 
		FOREIGN KEY(client_id) REFERENCES clients(id),
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id)
	);`,

	`CREATE TABLE audit_trails (
		id BIGSERIAL PRIMARY KEY,
		client_id INT NOT NULL,
		change TEXT NOT NULL,
		role_used TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(client_id) REFERENCES clients(id)
	);`,

	`CREATE TABLE client_order_history (
		id BIGSERIAL PRIMARY KEY,
		order_id TEXT NOT NULL,
		client_id INT NOT NULL,
		exchange_id INT NOT NULL,
		currency_pair TEXT NOT NULL,
		asset_type TEXT NOT NULL,
		order_type TEXT NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		rate DOUBLE PRECISION NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id),
		FOREIGN KEY(client_id) REFERENCES clients(id),
		UNIQUE(exchange_id, order_id)
	);`,

	`CREATE TABLE exchange_platform_trade_history (
		id BIGSERIAL PRIMARY KEY,
		order_id TEXT NOT NULL,
		exchange_id INT NOT NULL,
		currency_pair VARCHAR(20) NOT NULL,
		asset_type TEXT NOT NULL,
		order_type TEXT DEFAULT 'NOT SPECIFIED' NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		rate DOUBLE PRECISION NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id),
		UNIQUE(exchange_id, order_id)
	);`,
}
