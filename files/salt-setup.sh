source /etc/environment

sudo apt update

#Get the Minion Addr
wget https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/minion

sed -i 's/\$SALT_MASTER_HOST/'$SALT_MASTER_HOST'/' minion
sed -i 's/\$SALT_MASTER_FINGER/'$SALT_MASTER_FINGER'/' minion

sudo mv minion /etc/salt/minion

sudo setfacl -m u:pi:rwx /etc/salt/pki/minion/
sudo setfacl -m u:pi:rwx /etc/salt/pki/minion/*

sudo systemctl restart salt-minion
sudo touch /etc/salt/setup

# id
