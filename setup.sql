create database pingnstor;
create table pingnstor.pings
(
	ping_id bigint primary key AUTO_INCREMENT,
	the_date datetime not null default CURRENT_TIMESTAMP,
	domain varchar(4000) not null,
	packet_rtt float
)