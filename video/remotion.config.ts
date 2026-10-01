import { Config } from "@remotion/cli/config";
import path from "path";

Config.setVideoImageFormat("jpeg");
Config.setOverwriteOutput(true);
Config.setChromiumOpenGlRenderer("angle-egl");
Config.setConcurrency(2);
Config.setPublicDir(path.join(__dirname, "public"));
