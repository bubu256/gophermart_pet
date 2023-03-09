BEGIN;
CREATE TABLE IF NOT EXISTS users(
    user_id serial PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders(
    order_id serial PRIMARY KEY,
    number TEXT UNIQUE NOT NULL,
    user_id INT NOT NULL,
    CONSTRAINT fk_order_users FOREIGN KEY(user_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS status(
    status_id serial PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

INSERT INTO status(name)
VALUES ('NEW'), ('PROCESSING'), ('INVALID'), ('PROCESSED');

CREATE TABLE IF NOT EXISTS order_status(
    order_status_id serial PRIMARY KEY,
    status_id INT UNIQUE NOT NULL,
    order_id INT UNIQUE NOT NULL,
    datetime TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_order_status_status FOREIGN KEY(status_id) REFERENCES status(status_id),
    CONSTRAINT fk_order_status_orders FOREIGN KEY(order_id) REFERENCES orders(order_id)
);

CREATE TABLE IF NOT EXISTS bonus_flow(
    bonus_flow_id serial PRIMARY KEY,
    user_id INT NOT NULL,
    order_id INT,
    amount REAL NOT NULL,
    datetime TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_bonus_flow_users FOREIGN KEY(user_id) REFERENCES users(user_id),
    CONSTRAINT fk_bonus_flow_orders FOREIGN KEY(order_id) REFERENCES orders(order_id)
);

COMMIT;