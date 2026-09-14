create table if not exists users(
    id varchar(255) primary key,
    name varchar(255) not null,
    email varchar(255) unique not null,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);