# RE4 Biorand Reseed Tool

Not affiliated with Biorand in any way, use it at your own risk.

## What is this?
This is a small tool I wrote because downloading a new seed from Biorand every time I died and manually extracting it was annoying.

## How do I set it up?
Download the latest compiled binary from [here](https://github.com/mtfarkas/re4-biorand-reseed/releases) or compile it from source for your preferred platform/architecture.

If you downloaded the compiled binary, extract the ZIP file somewhere on your machine. You should end up with a folder structure like this:

![image](https://github.com/user-attachments/assets/31561d74-41e7-4e57-ab09-1ef4d0082458)

Edit `reseed-config.json`; you need to add the folder you installed RE4 Remake to in the quotes after `"RE4InstallPath"`. On Windows, you have to switch every `\` character in the path for `\\`.

Next, you have to acquire a token from Biorand and add it to the file in the quotes after `"BiorandToken"`. To do this, do the following:
1. Visit the [Official Biorand Website](https://re4r.biorand.net/)
2. Make sure you're logged in
3. Open your browser's devtools (F12 on Chrome/Firefox)
4. Paste the following into the console: `JSON.parse(localStorage.getItem("userManager"))?.token ?? "No token found, make sure you're logged in!"`
   - Disclaimer: you should never run random commands in your browser from the internet, so let's break this one down
   - Biorand puts your user information in your browser's local storage after logging in; To get the token we read this: `localStorage.getItem("userManager")`
   - We then parse this string of characters into a Javascript object with `JSON.parse()`
   - We then try to read the `.token` property of this parsed object; If successful, the console should spit out the token
   - If not, we instead print a message
5. Copy the token from the console and paste it between the quotes after `"BiorandToken"`

After you're done with everything, your config should look something like this:

![image](https://github.com/user-attachments/assets/976a4065-12e3-4be2-b7c2-acf58260645b)

## How do I use it?
If you completed the steps in the setup section, using the tool is as easy as just opening the `.exe` file you extracted.
1. The tool will always download the latest configuration from 7rayD's Balanced Combat Randomizer (this is hardcoded in the tool for now)
   - If this step fails, you have to log in to the [Official Biorand Website](https://re4r.biorand.net/) and bookmark the profile.
2. It will then generate a random 6-digit seed
3. It will ask you if you want to continue. Press `y` to continue. (note: the process is not cancellable after this and it will overwrite your previously installed seed)
4. If answering yes, the tool will queue a new seed for generation, wait for it to finish, download and extract it to your RE4 installation directory. Done!

Note: the tool will keep every seed generated with it in the `biorand-seeds` directory. This is useful if you want to go back to some older seeds you had.

## Troubleshooting
- If at any point you get an HTTP error 401, make sure you put your Biorand token in your config file correctly. If the problem persists, log out and log in to the Biorand website again and repeat the process of extracting your token and putting it in the config file.
- If you're having trouble downloading/extracting seed zips, make sure you have write access to both the directory the tool is in, and your RE4 install directory. As a last resort, you can run the tool as an administrator.

## License 
MIT
