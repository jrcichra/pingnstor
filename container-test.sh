docker network create test-net
docker run --name pingnstor --network="host" jrcichra/pingnstor:${TRAVIS_COMMIT}
sleep 120
mysql -uroot -sse "select count(*) from pingnstor.pings where domain = 'google.com'"
docker kill pingnstor