# Goal of this project 

To make a postgres db monitor with ui. (Desktop GUI app for monitoring postgres db)

I have postgres server running in my pc or somewhere in remote vps, i have diffculty monitoring it, like there is database, tables, many connections, i want to make it simple by just maintaining it via desktop GUI.

I can just remote `ssh system` and it will just get into that machine. and once it gets to machine it will auto download or rather upload i would say a remote binary which will have http points defined, so the local ui will just send signals via ssh to these https endpoints, and it will just monitor those.

For now i just kept it simple i have just added database, tables, and user can view their records. 

In future i want to monitor everything postgres has to offer, like if user want to see the speed of queries, cache miss or many different things user has to offer, we can just use postgres monitor to monitor those. Earlier i though about using ebpf for monitoring it to base level, but postgres 18 gave many option and it became a liability rather than asset to maintain ebpf, so i decided to switch to postgres one instead. 

Yes, I am not an expert in postgres, that is one thing, i will constantly try to learn along this journey. Almost everything from **first principles**, I have build a db from scratch once(yet to be uploaded on github becoz it is not complelete yet), although it was basic one, but it gave me lots of insight on how we can build this from scratch.

I want to also mention that i dont like abstractions, it's just blocks my brain for no reason, and i don't have to do manual things now like with the help of ai, mostly the frontend and wails part ai will handle, for me frontend is diffcult i have to mention it, people say its easy, i genuinely believe its difficult atleast for me but since the advent of ai, i can just focus on core principles. Also thanks to **Ben Dickens**, what an amazing teacher he is. His db course is literally the best course on db i have seen till now.

# Features shipped
