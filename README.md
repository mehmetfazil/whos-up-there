# Who's Flying Over

Some days I recognized there are lot's of planes, some days hardly any. I doubt it is due to wind direction, so airports are using runways on the other way around. So I had to find out. Also, isn't it cool to have a departure for, just for yourself? 

I will build a Go application, which will use public API's to fetch the flight info, with aircraft and route details. The frontend will be easy, no need to use a bloated javascript framework, instead I'll rely on HTMX and Tailwind (and ChatGPT of course).  
Backend will be a data service, written in Go, raw SQL, no frameworks or ORMs, and will store the data in a local postgres db.