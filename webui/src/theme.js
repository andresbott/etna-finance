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
                    50: '#faf9f5',
                    100: '#f5f3ec',
                    200: '#ebe7db',
                    300: '#dcd6c5',
                    400: '#c4baa7',
                    500: '#a89b85',
                    600: '#8a7b66',
                    700: '#6b5f4d',
                    800: '#4d4338',
                    900: '#2f2a22',
                    950: '#1a1612'
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
                    0: '#ffffff',
                    50: '#faf9f5',
                    100: '#f5f3ec',
                    200: '#ebe7db',
                    300: '#dcd6c5',
                    400: '#c4baa7',
                    500: '#a89b85',
                    600: '#8a7b66',
                    700: '#6b5f4d',
                    800: '#4d4338',
                    900: '#2f2a22',
                    950: '#1a1612'
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
    }
})

export default CustomTheme
