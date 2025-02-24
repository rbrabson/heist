// Utility routine to convert between the economy file used by Yeti and the one used by Heist.

package main

type OldEconomy interface{}

func main() {
	/*
		economy.Start(nil)
		fmt.Println("Getting accounts")
		start := time.Now()
		accounts := economy.GetMonthlyLeaderboard("1141342869383282759", 10)
		elapsed := time.Since(start)
		fmt.Printf("Elapsed time sorting accounts: %s\n", elapsed)
		for _, account := range accounts {
			fmt.Printf("Name: %s, Balance: %d\n", account.Name, account.CurrentBalance)
		}

		start = time.Now()
		ranking := economy.GetMonthlyRanking("1133473421532086342", "149670432331661312")
		elapsed = time.Since(start)
		fmt.Printf("Elapsed time getting ranking: %s\n", elapsed)
		fmt.Printf("Ranking: %d\n", ranking)
	*/

	/*
		godotenv.Load()
		economy.LoadBanks()
		fmt.Println("Number of banks:", len(banks))
		for _, bank := range banks {
			fmt.Printf("Bank %s has %d accounts\n", bank.ID, len(bank.Accounts))
			account := bank.Accounts["149670432331661312"]
			fmt.Printf("%s's account has %d %s\n", account.Name, account.Balance, bank.Currency)
		}
	*/
	/*

		economy.Banks = make(map[string]*economy.Bank)

		oldFilename := "./store/economy/old_economy.json"

		data, err := os.ReadFile(oldFilename)
		if err != nil {
			log.Fatal("Unable to open economy json file, error:", err)
		}
		var oldEconomy OldEconomy
		err = json.Unmarshal(data, &oldEconomy)
		if err != nil {
			log.Fatal("Unable to unmarshal economy data, error:", err)
		}

		top, ok := oldEconomy.(map[string]interface{})
		if !ok {
			log.Fatal("Can't get the oldEconomy as a map")
		}
		var bank economy.Bank
		for k := range top {
			bank = *economy.GetBank(k)
			break
		}
		bank.DefaultBalance = 20000
		second := top[bank.ID].(map[string]interface{})
		var _, guilds, guildMembers map[string]interface{}
		for k, v := range second {
			switch k {
			case "GLOBAL":
			case "GUILD":
				guilds = v.(map[string]interface{})
			case "MEMBER":
				guildMembers = v.(map[string]interface{})
			case "USER":
			default:
				log.Error("Unknown key ", k)
			}
		}

		for _, v := range guilds {
			data := v.(map[string]interface{})

			for k2, v2 := range data {
				switch k2 {
				case "bank_name":
					bank.BankName = v2.(string)
				case "currency":
					bank.Currency = v2.(string)
				case "default_balance":
					bank.DefaultBalance = int(v2.(float64))
				default:
					log.Error("Unknown key ", k2)
				}
			}
		}

		for _, v := range guildMembers {
			data := v.(map[string]interface{})

			for k2, v2 := range data {
				memberData := v2.(map[string]interface{})
				account := &economy.Account{
					ID: k2,
				}

				for k3, v3 := range memberData {
					switch k3 {
					case "balance":
						account.CurrentBalance = int(v3.(float64))
						account.LifetimeBalance = account.CurrentBalance
					case "created_at":
						epoch := int64(v3.(float64))
						account.CreatedAt = time.Unix(epoch, 0)
					case "name":
						account.Name = v3.(string)
					default:
						log.Error("Unknown key ", k3)
					}
				}

				bank.Accounts[k2] = account
			}
		}

		newFilename := "./store/economy/economy.json"
		// write the new economy to the file system
		data, err = json.Marshal(bank)
		if err != nil {
			log.Fatal("Unable to unmarshal the new economy, error:", err)
		}
		err = os.WriteFile(newFilename, data, 0644)
		if err != nil {
			log.Fatal("Unable to write the new economy file, error:", err)
		}
	*/
}
