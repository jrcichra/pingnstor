if [ $(mysql -uroot -sse "select count(*) from pingnstor.pings where domain = 'google.com'") -gt 0 ]
then
exit 0
else
exit 1
fi