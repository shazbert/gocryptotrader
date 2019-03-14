package postgres

var postgresSchema = map[string]string{
	"client": `CREATE TABLE client (
		id SERIAL PRIMARY KEY,
  		user_name VARCHAR(50) NOT NULL,
		password VARCHAR(50) NOT NULL,
		email VARCHAR(50),
		role VARCHAR(50),
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		last_logged_in TIMESTAMP NOT NULL,
		UNIQUE(user_name)
	  );`,

	"exchange": `CREATE TABLE exchange (
		id SERIAL PRIMARY KEY,
		exchange_name VARCHAR(50) NOT NULL,
		created_at TIMESTAMP NOT NULL,
		UNIQUE(exchange_name)
	);`,

	"client_order_history": `CREATE TABLE client_order_history (
		id SERIAL PRIMARY KEY,
		order_id VARCHAR(50) NOT NULL,
		currency_pair VARCHAR(20) NOT NULL,
		asset_type VARCHAR(10) NOT NULL,
		order_type VARCHAR(10) NOT NULL,
		amount REAL NOT NULL,
		rate REAL NOT NULL,
		exchange_id SMALLINT NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchange(id),
		UNIQUE(exchange_id, order_id)
	);`,

	"exchange_platform_trade_history": `CREATE TABLE exchange_platform_trade_history (
		id SERIAL PRIMARY KEY,
		order_id VARCHAR(50) NOT NULL,
		currency_pair VARCHAR(20) NOT NULL,
		asset_type VARCHAR(10) NOT NULL,
		order_type VARCHAR(10) DEFAULT 'NOT SPECIFIED' NOT NULL,
		amount REAL NOT NULL,
		rate REAL NOT NULL,
		exchange_id SMALLINT NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchange(id),
		UNIQUE(exchange_id, order_id)
	);`}
