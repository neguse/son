module Main exposing (..)

import Array exposing (Array, empty, toList)
import Keyboard exposing (..)
import Html exposing (..)
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
    { l : Bool
    , r : Bool
    , u : Bool
    }


type alias Model =
    { inputUser : String
    , inputBody : String
    , state : S2CMessage
    , keyState : C2SMessage
    }


init : ( Model, Cmd Msg )
init =
    ( Model "" "" (S2CMessage empty) (C2SMessage False False False), Cmd.none )


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
        [ ( "l", Json.Encode.bool msg.l )
        , ( "r", Json.Encode.bool msg.r )
        , ( "u", Json.Encode.bool msg.u )
        ]



-- UPDATE


type Msg
    = InputUser String
    | InputBody String
    | NewMessage String
    | KeyDown KeyCode
    | KeyUp KeyCode


leftKeyCode : Int
leftKeyCode =
    37


rightKeyCode : Int
rightKeyCode =
    39


upKeyCode : Int
upKeyCode =
    38


changeKey : KeyCode -> Bool -> C2SMessage -> C2SMessage
changeKey key isDown state =
    if key == leftKeyCode then
        { state | l = isDown }
    else if key == rightKeyCode then
        { state | r = isDown }
    else if key == upKeyCode then
        { state | u = isDown }
    else
        state


update : Msg -> Model -> ( Model, Cmd Msg )
update msg { inputUser, inputBody, state, keyState } =
    case msg of
        InputUser newUser ->
            ( Model newUser inputBody state keyState, Cmd.none )

        InputBody newBody ->
            ( Model inputUser newBody state keyState, Cmd.none )

        NewMessage str ->
            case decodeString messageDecoder str of
                Ok newState ->
                    ( Model inputUser inputBody newState keyState, Cmd.none )

                Err err ->
                    Debug.crash err

        KeyDown code ->
            let
                newKeyState =
                    (changeKey code True keyState)
            in
                ( Model inputUser inputBody state newKeyState
                , WebSocket.send serverAddr (encode 0 (messageEncoder newKeyState))
                )

        KeyUp code ->
            let
                newKeyState =
                    (changeKey code False keyState)
            in
                ( Model inputUser inputBody state newKeyState
                , WebSocket.send serverAddr (encode 0 (messageEncoder newKeyState))
                )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ WebSocket.listen serverAddr NewMessage
        , downs KeyDown
        , ups KeyUp
        ]



-- VIEW


view : Model -> Html Msg
view model =
    svg [ width "320", height "320", viewBox "0 0 320 320" ]
        ((rect [ width "320", height "320", fill "none", stroke "#000" ] [])
            :: (List.concatMap viewPlayer (toList model.state.players))
        )


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
