// Utility routine to convert between the economy file used by Yeti and the one used by Heist.

package main

type OldEconomy interface{}

func main() {
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

			e := economy.Economy{}
			e.Guilds = make(map[string]economy.Guild)
			e.GuildMembers = make(map[string]economy.GuildMembers)

			top, ok := oldEconomy.(map[string]interface{})
			if !ok {
				log.Fatal("Can't get the oldEconomy as a map")
			}
			for k := range top {
				e.ID = k
			}
			second := top[e.ID].(map[string]interface{})
			var global, guilds, guildMembers map[string]interface{}
			for k, v := range second {
				switch k {
				case "global":
					global = v.(map[string]interface{})
				case "guilds":
					guilds = v.(map[string]interface{})
				case "members":
					guildMembers = v.(map[string]interface{})
				case "users":
				default:
					log.Error("Unknown key ", k)
				}
			}

			e.Global.SchemaVersion = int(global["schema_version"].(float64))

			for k, v := range guilds {
				data := v.(map[string]interface{})
				guild := economy.Guild{
					ID: k,
				}

				for k2, v2 := range data {
					switch k2 {
					case "bank_name":
						guild.BankName = v2.(string)
					case "currency":
						guild.Currency = v2.(string)
					case "default_balance":
						guild.DefaultBalance = int(v2.(float64))
					default:
						log.Error("Unknown key ", k2)
					}
				}

				e.Guilds[k] = guild
			}

			for k, v := range guildMembers {
				data := v.(map[string]interface{})

				guildMembers := economy.GuildMembers{
					ID: k,
				}
				guildMembers.Members = make(map[string]economy.Member)

				for k2, v2 := range data {
					memberData := v2.(map[string]interface{})
					member := economy.Member{
						ID: k2,
					}

					for k3, v3 := range memberData {
						switch k3 {
						case "balance":
							member.Balance = int(v3.(float64))
						case "created_at":
							epoch := int64(v3.(float64))
							member.CreatedAt = time.Unix(epoch, 0)
						case "name":
							member.Name = v3.(string)
						default:
							log.Error("Unknown key ", k3)
						}
					}

					guildMembers.Members[k2] = member
				}

				e.GuildMembers[k] = guildMembers
			}

			newFilename := "./store/economy/economy.json"
			// write the new economy to the file system
			data, err = json.Marshal(e)
			if err != nil {
				log.Fatal("Unable to unmarshal the new economy, error:", err)
			}
			err = os.WriteFile(newFilename, data, 0644)
			if err != nil {
				log.Fatal("Unable to write the new economy file, error:", err)
			}
	*/
}
