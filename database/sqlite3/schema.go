package sqlite3

var sqliteSchema = map[string]string{
	"client": `CREATE TABLE client (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  		user_name text NOT NULL,
		password text NOT NULL,
		email text,
		role text,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_logged_in DATETIME NOT NULL,
		UNIQUE(user_name)
	  );`,

	"exchange": `CREATE TABLE exchange (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		exchange_name text NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE(exchange_name)
	);`,

	"client_order_history": `CREATE TABLE client_order_history (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		order_id text NOT NULL,
		fulfilled_on DATETIME NOT NULL,
		currency_pair text NOT NULL,
		asset_type text NOT NULL,
		order_type text NOT NULL,
		amount real NOT NULL,
		rate real NOT NULL,
		exchange_id int NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchange(id),
		UNIQUE(exchange_id, order_id)
	);`,

	"exchange_platform_trade_history": `CREATE TABLE exchange_platform_trade_history (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		fulfilled_on DATETIME NOT NULL,
		currency_pair text NOT NULL,
		asset_type text NOT NULL,
		order_type text NOT NULL DEFAULT "NOT SPECIFIED",
		amount real NOT NULL,
		rate real NOT NULL,
		order_id text NOT NULL,
		exchange_id int NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchange(id),
		UNIQUE(exchange_id, order_id)
	);`}

var deprecatedDatabaseTables = []string{}
