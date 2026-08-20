#import <Foundation/Foundation.h>

static NSMutableDictionary *mutableDictionary(NSDictionary *source) {
    return source ? [source mutableCopy] : [NSMutableDictionary dictionary];
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 3 || strcmp(argv[1], "set-jmbe") != 0) {
            fprintf(stderr, "usage: gpsdr-mac-prefs set-jmbe /path/to/jmbe.jar\n");
            return 2;
        }
        NSString *jmbePath = [NSString stringWithUTF8String:argv[2]];
        if (!jmbePath.length || ![[NSFileManager defaultManager] isReadableFileAtPath:jmbePath]) {
            fprintf(stderr, "JMBE library is not readable\n");
            return 3;
        }

        NSString *domain = @"io.github.dsheirer";
        NSUserDefaults *defaults = [NSUserDefaults standardUserDefaults];
        NSMutableDictionary *persistent = mutableDictionary([defaults persistentDomainForName:domain]);
        NSMutableDictionary *root = mutableDictionary(persistent[@"/io/github/dsheirer/"]);
        NSMutableDictionary *preference = mutableDictionary(root[@"preference/"]);
        NSMutableDictionary *decoder = mutableDictionary(preference[@"decoder/"]);
        decoder[@"path.jmbe.library.1.0.0"] = jmbePath;
        preference[@"decoder/"] = decoder;
        root[@"preference/"] = preference;
        persistent[@"/io/github/dsheirer/"] = root;
        [defaults setPersistentDomain:persistent forName:domain];
        if (![defaults synchronize]) {
            fprintf(stderr, "could not synchronize SDRTrunk preferences\n");
            return 4;
        }
    }
    return 0;
}
