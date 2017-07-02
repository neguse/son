module Main exposing (..)

import Array exposing (Array, empty, toList)
import Color exposing (Color)
import Html exposing (..)
import Html.Events exposing (..)
import Html.Attributes exposing (value)
import WebSocket
import Json.Encode exposing (Value, encode)
import Json.Decode exposing (decodeString, Decoder)
import Json.Decode.Pipeline exposing (decode, required)
import Svg exposing (..)
import Svg.Attributes exposing (..)


serverAddr : String
serverAddr =
    "ws://localhost:12345/ws"


main : Program Never Model Msg
main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }



-- MODEL


type alias S2CMessage =
    { players : Array Player }


type alias Player =
    { x : Float
    , y : Float
    , a : Float
    , r : Float
    }


type alias C2SMessage =
    { user : String
    , body : String
    }


type alias Model =
    { inputUser : String
    , inputBody : String
    , state : S2CMessage
    }


init : ( Model, Cmd Msg )
init =
    ( Model "" "" (S2CMessage empty), Cmd.none )


playerDecoder : Decoder Player
playerDecoder =
    decode Player
        |> required "x" Json.Decode.float
        |> required "y" Json.Decode.float
        |> required "a" Json.Decode.float
        |> required "r" Json.Decode.float


messageDecoder : Decoder S2CMessage
messageDecoder =
    decode S2CMessage
        |> required "players" (Json.Decode.array playerDecoder)


messageEncoder : C2SMessage -> Value
messageEncoder msg =
    Json.Encode.object
        [ ( "user", Json.Encode.string msg.user )
        , ( "body", Json.Encode.string msg.body )
        ]



-- UPDATE


type Msg
    = InputUser String
    | InputBody String
    | Send
    | NewMessage String


update : Msg -> Model -> ( Model, Cmd Msg )
update msg { inputUser, inputBody, state } =
    case msg of
        InputUser newUser ->
            ( Model newUser inputBody state, Cmd.none )

        InputBody newBody ->
            ( Model inputUser newBody state, Cmd.none )

        Send ->
            ( Model inputUser "" state
            , WebSocket.send serverAddr
                (encode 0 (messageEncoder (C2SMessage inputUser inputBody)))
            )

        NewMessage str ->
            case decodeString messageDecoder str of
                Ok newState ->
                    ( Model inputUser inputBody newState, Cmd.none )

                Err err ->
                    ( Model inputUser inputBody state, Debug.crash err )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    WebSocket.listen serverAddr NewMessage



-- VIEW


view : Model -> Html Msg
view model =
    svg [ width "320", height "320", viewBox "0 0 320 320" ]
        ((rect [ width "320", height "320", fill "none", stroke "#000" ] [])
           :: (List.concatMap viewPlayer (toList model.state.players)))
        
viewPlayer : Player -> List (Svg msg)
viewPlayer p =
    [ circle
        [ cx (toString p.x)
        , cy (toString p.y)
        , r (toString p.r)
        , stroke "#000"
        ]
        []
    , line
        [ x1 (toString p.x)
        , y1 (toString p.y)
        , x2 (toString (p.x + p.r * 2 * (cos p.a)))
        , y2 (toString (p.y + p.r * 2 * (sin p.a)))
        , stroke "#000"
        ]
        []
    ]
