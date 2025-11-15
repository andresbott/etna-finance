import { definePreset } from '@primevue/themes'
import Lara from '@primevue/themes/lara'

const CustomTheme = definePreset(Lara, {
    primitive: {
        borderRadius: {
            none: '0',
            xs: '1px',
            sm: '1px',
            md: '1px',
            lg: '1px',
            xl: '1px'
        },
        // Custom warm, earthy palette with dramatic accents
        autumn: {
            50: '#fff8e5',
            100: '#fff3b0',
            200: '#f4d98f',
            300: '#e09f3e',
            400: '#c88534',
            500: '#9e6b2f',
            600: '#7a4e2a',
            700: '#9e2a2b',
            800: '#6b1d1e',
            900: '#540b0e',
            950: '#2a0507'
        },
        slate: {
            50: '#e8f0f2',
            100: '#c5dade',
            200: '#9ebfc7',
            300: '#6a99a5',
            400: '#4e7d8b',
            500: '#335c67',
            600: '#2a4d57',
            700: '#203b43',
            800: '#162a30',
            900: '#0d1a1e',
            950: '#070f11'
        }
    },
    semantic: {
        primary: {
            50: '#e8f0f2',
            100: '#c5dade',
            200: '#9ebfc7',
            300: '#6a99a5',
            400: '#4e7d8b',
            500: '#335c67',
            600: '#2a4d57',
            700: '#203b43',
            800: '#162a30',
            900: '#0d1a1e',
            950: '#070f11'
        },
        colorScheme: {
            light: {
                primary: {
                    color: '{primary.500}',
                    contrastColor: '#ffffff',
                    hoverColor: '{primary.600}',
                    activeColor: '{primary.700}'
                },
                highlight: {
                    background: '#fff3b0',
                    focusBackground: '#f4d98f',
                    color: '#540b0e',
                    focusColor: '#9e2a2b'
                },
                surface: {
                    0: '#ffffff',
                    50: '#fafafa',
                    100: '#f5f5f5',
                    200: '#e5e5e5',
                    300: '#d4d4d4',
                    400: '#a3a3a3',
                    500: '#737373',
                    600: '#525252',
                    700: '#404040',
                    800: '#262626',
                    900: '#171717',
                    950: '#0a0a0a'
                }
            },
            dark: {
                primary: {
                    color: '{primary.400}',
                    contrastColor: '#ffffff',
                    hoverColor: '{primary.300}',
                    activeColor: '{primary.200}'
                },
                highlight: {
                    background: 'rgba(224, 159, 62, .16)',
                    focusBackground: 'rgba(224, 159, 62, .24)',
                    color: 'rgba(255, 243, 176, .87)',
                    focusColor: 'rgba(255, 243, 176, .87)'
                },
                surface: {
                    0: '#0a0a0a',
                    50: '#0a0a0a',
                    100: '#171717',
                    200: '#262626',
                    300: '#404040',
                    400: '#525252',
                    500: '#737373',
                    600: '#a3a3a3',
                    700: '#d4d4d4',
                    800: '#e5e5e5',
                    900: '#f5f5f5',
                    950: '#fafafa'
                }
            }
        },
        focusRing: {
            width: '2px',
            style: 'solid',
            color: '#e09f3e',
            offset: '2px',
            shadow: '0 0 0 0.2rem rgba(224, 159, 62, 0.5)'
        }
    },
    components: {
        card: {
            colorScheme: {
                light: {
                    root: {
                        background: '{surface.0}',
                        color: '{surface.700}'
                    }
                },
                dark: {
                    root: {
                        background: '{surface.0}',
                        color: '{surface.200}'
                    }
                }
            }
        },
        panel: {
            colorScheme: {
                light: {
                    root: {
                        background: '{surface.0}'
                    }
                },
                dark: {
                    root: {
                        background: '{surface.0}'
                    }
                }
            }
        }
    }
})

export default CustomTheme
