#!/bin/sh

go build -o C:/Dev/AUC-DEMO/provider0 C:/Dev/AUC/wifi-trade-consensus/cmd/provider
go build -o C:/Dev/AUC-DEMO/provider1 C:/Dev/AUC/wifi-trade-consensus/cmd/provider
go build -o C:/Dev/AUC-DEMO/provider2 C:/Dev/AUC/wifi-trade-consensus/cmd/provider

# wt --title "provider0" C:/Dev/AUC-DEMO/provider0/provider &
# wt new-tab -w 0 --title "provider1" C:/Dev/AUC-DEMO/provider1/provider &
# wt -w 0 --title "provider2" C:/Dev/AUC-DEMO/provider2/provider &