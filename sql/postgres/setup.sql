create table pings
(
	ping_id BIGSERIAL,
	the_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	domain varchar(4000) not null,
	packet_rtt float,
	next_hop boolean not null
)